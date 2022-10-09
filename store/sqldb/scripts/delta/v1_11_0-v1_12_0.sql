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

CREATE TABLE `routing_config_v2`
(
    `id`       VARCHAR(128) NOT NULL comment 'ID',
    `name`     VARCHAR(64) NOT NULL,
    `namespace`     VARCHAR(64) NOT NULL,
    `policy`   VARCHAR(64) NOT NULL,
    `config`   TEXT,
    `enable`   INT         NOT NULL DEFAULT 0,
    `revision` VARCHAR(40) NOT NULL,
    `description` VARCHAR(500) NOT NULL DEFAULT '',
    `priority`   smallint(6)    NOT NULL DEFAULT '0' comment 'ratelimit rule priority',
    `flag`     TINYINT(4)  NOT NULL DEFAULT '0',
    `ctime`    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `etime`    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `extend_info` VARCHAR(1024) DEFAULT '',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) engine = innodb;

