#!/bin/sh

# fail if some commands fails
set -e
# show commands
set -x

pip install pika --upgrade
python scripts/prow.py
exit 1
