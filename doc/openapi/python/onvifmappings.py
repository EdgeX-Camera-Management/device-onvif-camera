#!/usr/bin/env python3
#
# Copyright (C) 2022-2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#


from dataclasses import dataclass, field
import sys
from typing import Dict
import logging
import os

from ruamel.yaml import YAML


yaml = YAML()
log = logging.getLogger('onvifmapping')

EDGEX = 'EdgeX'

# mapping of service name to wsdl file for externalDocs
SERVICE_WSDL = {
    'Analytics': 'https://www.onvif.org/ver20/analytics/wsdl/analytics.wsdl',
    'Device': 'https://www.onvif.org/ver10/device/wsdl/devicemgmt.wsdl',
    'Event': 'https://www.onvif.org/ver10/events/wsdl/event.wsdl',
    'Imaging': 'https://www.onvif.org/ver20/imaging/wsdl/imaging.wsdl',
    'Media': 'https://www.onvif.org/ver10/media/wsdl/media.wsdl',
    'Media2': 'https://www.onvif.org/ver20/media/wsdl/media.wsdl',
    'PTZ': 'https://www.onvif.org/ver20/ptz/wsdl/ptz.wsdl'
}

HEADER = """\
| REST | EdgeX Command                    | Onvif Service | Onvif Function                      | 
|------|----------------------------------|---------------|-------------------------------------|
"""


@dataclass
class OnvifMapper:
    profile_file: str
    output_file: str

    yml: any = None
    profile: any = None
    resources: Dict[str, any] = field(default_factory=dict)

    def _load(self):
        """Read input yaml files"""
        log.info(f'Reading profile file: {self.profile_file}')
        with open(self.profile_file) as f:
            self.profile = yaml.load(f)

    def _parse(self):
        """Parse the device resources into a lookup table"""
        for resource in self.profile['deviceResources']:
            self.resources[resource['name']] = resource

    def _write(self):
        """Output modified yaml file"""
        log.info(f'Writing output file: {self.output_file}')
        with open(self.output_file, 'w') as w:
            w.write(HEADER)
            """Maps all functions from the device profile to their Onvif counterparts"""
            for cmd, cmd_obj in self.resources.items():
                if cmd_obj['isHidden'] is True:
                    continue  # skip hidden commands (not callable by the core-command service)

                if 'getFunction' in cmd_obj['attributes']:
                    self._write_mapping(w, 'GET', cmd, cmd_obj, 'getFunction')

                if 'setFunction' in cmd_obj['attributes']:
                    self._write_mapping(w, 'PUT', cmd, cmd_obj, 'setFunction')

    @staticmethod
    def _write_mapping(w, rest_method, cmd, cmd_obj, func_type):
        service = cmd_obj['attributes']['service']
        fn = cmd_obj['attributes'][func_type]
        if service == EDGEX:
            w.write(f"| {rest_method} | [{cmd}]() | EdgeX Custom |  |\n")
        else:
            w.write(f"| {rest_method} | [{cmd}]() | [{service}]({SERVICE_WSDL[service]}) | [{fn}]({SERVICE_WSDL[service]}#op.{fn}) |\n")

    def process(self):
        """Process the input yaml files, and create the final output yaml file"""
        self._load()
        self._parse()
        self._write()


def main():
    if len(sys.argv) != 3:
        print(f'Usage: {sys.argv[0]} <profile_file> <output_file>')
        sys.exit(1)

    logging.basicConfig(level=(logging.DEBUG if os.getenv('DEBUG_LOGGING') == '1' else logging.INFO),
                        format='%(asctime)-15s %(levelname)-8s %(name)-12s %(message)s')

    proc = OnvifMapper(sys.argv[1],  # profile_file
                       sys.argv[2])  # output_file
    proc.process()


if __name__ == '__main__':
    main()
