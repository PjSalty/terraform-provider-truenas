#!/usr/bin/env bash
# truenas_system_update is a singleton — the only valid import ID is the
# literal string "system_update".
terraform import truenas_system_update.prod system_update
