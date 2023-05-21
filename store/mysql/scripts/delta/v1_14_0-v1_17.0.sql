/*
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */
--
-- Database: `polaris_server`
--
USE `polaris_server`;

ALTER TABLE instance_metadata DROP FOREIGN KEY `instance_metadata_ibfk_1`;
ALTER TABLE service_metadata DROP FOREIGN KEY `service_metadata_ibfk_1`;
ALTER TABLE health_check DROP FOREIGN KEY `health_check_ibfk_1`;
ALTER TABLE circuitbreaker_rule_relation DROP FOREIGN KEY `circuitbreaker_rule_relation_ibfk_1`;
