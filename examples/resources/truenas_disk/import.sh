#!/usr/bin/env bash
# The import ID is the disk identifier, as shown by the truenas_disk data
# source. Quote it: it starts with a brace.
terraform import truenas_disk.bay3 '{serial_lunid}WD-WCC4N1234567_5000cca2'
