#!/usr/bin/env python3
"""this script is used to parse the yaml file and generate the .env file."""

import sys
import yaml

# store irst python argument in variable
platform_selected = sys.argv[1]
file_env = f'.env-{platform_selected}'
env_prefix = 'NODE'

if platform_selected == 'stack':
    env_prefix = 'STACK'

PLATFORMS_FILE = '.e2e-platforms.yaml'
FILE_ENCODING = 'UTF-8'

with open(PLATFORMS_FILE, 'r', encoding=FILE_ENCODING) as stream:
    try:
        values = yaml.safe_load(stream)
        platforms = values['PLATFORMS']
        platform = platforms.get(platform_selected)
        if platform is None:
            print(f'Platform "{platform_selected}" not found')
            sys.exit(1)
        shell_type = platform.get('shell_type')
        if shell_type is None:
            shell_type = 'sh'
        image = platform.get('image')
        instance_type = platform.get('instance_type')
        user = platform.get('username')
        with open(file_env, 'w', encoding=FILE_ENCODING) as f:
            f.write(f"export {env_prefix}_IMAGE={image}\n")
            f.write(f"export {env_prefix}_INSTANCE_TYPE={instance_type}\n")
            f.write(f"export {env_prefix}_LABEL={platform_selected}\n")
            f.write(f"export {env_prefix}_SHELL_TYPE={shell_type}\n")
            f.write(f"export {env_prefix}_USER={user}\n")
    except yaml.YAMLError as exc:
        print("Error parsing YAML file: ", exc)
        sys.exit(1)
