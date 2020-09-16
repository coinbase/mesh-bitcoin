#!/bin/bash
# Copyright 2020 Coinbase, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

NETWORK=$1;
NAMESPACE=$2;
DATA_DIRECTORY=$3;
MAX_ITEMS=150000;

DATA_PATH="${DATA_DIRECTORY}/indexer";
DICT_PATH="assets/${NETWORK}-${NAMESPACE}.zstd";

rosetta-cli utils:train-zstd "${NAMESPACE}" "${DATA_PATH}" "${DICT_PATH}" "${MAX_ITEMS}" "${DICT_PATH}";
