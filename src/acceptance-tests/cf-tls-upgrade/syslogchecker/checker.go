package syslogchecker

import (
	"acceptance-tests/testing/helpers"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

type GuidGenerator interface {
	Generate() string
}

type Checker struct {
	syslogAppPrefix string
	guidGenerator   GuidGenerator
	errors          helpers.ErrorSet
	doneChn         chan struct{}
	retryAfter      time.Duration
}

func New(syslogAppPrefix string, guidGenerator GuidGenerator, retryAfter time.Duration) Checker {
	return Checker{
		syslogAppPrefix: syslogAppPrefix,
		guidGenerator:   guidGenerator,
		errors:          helpers.ErrorSet{},
		doneChn:         make(chan struct{}),
		retryAfter:      retryAfter,
	}
}

func (c Checker) Start(logSpinnerApp string, logSpinnerAppURL string) chan bool {
	watcher := make(chan bool)

	go func() {
		timer := time.After(0 * time.Second)
		for {
			select {
			case <-c.doneChn:
				close(watcher)
				return
			case <-timer:
				err := c.deploySyslogAndValidate(logSpinnerApp, logSpinnerAppURL)
				if err != nil {
					c.errors.Add(err)
				}
				watcher <- true

				timer = time.After(c.retryAfter)
			}
		}
	}()

	return watcher
}

func (c Checker) Stop() error {
	close(c.doneChn)
	return nil
}

func (c Checker) deploySyslogAndValidate(logSpinnerApp string, logSpinnerAppURL string) error {
	syslogDrainAppName := fmt.Sprintf("%s-%s", c.syslogAppPrefix, c.guidGenerator.Generate())

	err := setupSyslogDrainerApp(syslogDrainAppName)
	if err != nil {
		return err
	}

	session := cf.Cf("logs", syslogDrainAppName, "--recent").Wait()
	if session.ExitCode() != 0 {
		panic("could not start the syslog-drainer")
	}

	address := getSyslogAddress(session.Out.Contents())

	session = cf.Cf("cups", fmt.Sprintf("%s-service", syslogDrainAppName), "-l", fmt.Sprintf("syslog://%s", address)).Wait()
	if session.ExitCode() != 0 {
		panic("could not create the logger service")
	}

	session = cf.Cf("bind-service", logSpinnerApp, fmt.Sprintf("%s-service", syslogDrainAppName)).Wait()
	if session.ExitCode() != 0 {
		panic("could not bind the logger to the application")
	}

	session = cf.Cf("restage", logSpinnerApp).Wait()
	if session.ExitCode() != 0 {
		panic("could not restage the app")
	}

	guid := c.guidGenerator.Generate()
	err = sendGetRequestToApp(fmt.Sprintf("%s/log/%s", logSpinnerAppURL, guid))
	if err != nil {
		return err
	}

	session = cf.Cf("logs", syslogDrainAppName, "--recent").Wait()
	if session.ExitCode() != 0 {
		panic("could not get the logs for syslog drainer app")
	}

	err = validateDrainerGotGuid(string(session.Out.Contents()), guid)
	if err != nil {
		return err
	}

	err = cleanup(logSpinnerApp, syslogDrainAppName)
	if err != nil {
		panic(err)
	}

	return nil
}

func cleanup(logSpinnerApp, appName string) error {
	session := cf.Cf("unbind-service", logSpinnerApp, fmt.Sprintf("%s-service", appName)).Wait()
	if session.ExitCode() != 0 {
		panic(errors.New("could not unbind the logger to the application"))
	}

	session = cf.Cf("delete-service", fmt.Sprintf("%s-service", appName), "-f").Wait()
	if session.ExitCode() != 0 {
		panic(errors.New("could not delete the service"))
	}

	session = cf.Cf("delete", appName, "-f", "-r").Wait()
	if session.ExitCode() != 0 {
		panic(errors.New("could not delete the syslog drainer app"))
	}
	return nil
}

func setupSyslogDrainerApp(syslogDrainerAppName string) error {
	session := cf.Cf("push", syslogDrainerAppName, "-f", "assets/syslog-drainer/manifest.yml", "--no-start").Wait()
	if session.ExitCode() != 0 {
		return errors.New("syslog drainer application push failed")
	}

	session = cf.Cf("enable-diego", syslogDrainerAppName).Wait()
	if session.ExitCode() != 0 {
		panic(errors.New("could not enable diego for the syslog-drainer app"))
	}

	session = cf.Cf("start", syslogDrainerAppName).Wait()
	if session.ExitCode() != 0 {
		panic(errors.New("could not start the syslog-drainer app"))
	}
	return nil
}

func (c Checker) Check() error {
	if len(c.errors) > 0 {
		return c.errors
	}
	return nil
}

func getSyslogAddress(output []byte) string {
	re, err := regexp.Compile("ADDRESS: \\|(.*)\\|")
	if err != nil {
		panic(err)
	}
	address := re.FindSubmatch(output)[1]

	return string(address)
}

func sendGetRequestToApp(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

func validateDrainerGotGuid(logContents string, guid string) error {
	if !strings.Contains(logContents, guid) {
		return errors.New("could not validate the guid on syslog")
	}

	return nil
}
