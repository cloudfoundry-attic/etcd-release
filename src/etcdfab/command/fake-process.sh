#!/bin/sh
sleep 1
echo "Script with pid $$"
for i in 1 2 3; do
        sleep 1
        echo "Process running with pid $$"
done
