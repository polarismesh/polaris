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
CREATE
    DATABASE IF NOT EXISTS `polaris_server` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;

USE
    `polaris_server`;

CREATE TABLE `client`
(
    `id`      VARCHAR(128) NOT NULL comment 'client id',
    `host`    VARCHAR(100) NOT NULL comment 'client host IP',
    `type`    VARCHAR(100) NOT NULL comment 'client type: polaris-java/polaris-go',
    `version` VARCHAR(32)  NOT NULL comment 'client SDK version',
    `region`  varchar(128)          DEFAULT NULL comment 'region info for client',
    `zone`    varchar(128)          DEFAULT NULL comment 'zone info for client',
    `campus`  varchar(128)          DEFAULT NULL comment 'campus info for client',
    `flag`    tinyint(4)   NOT NULL DEFAULT '0' COMMENT '0 is valid, 1 is invalid(deleted)',
    `ctime`   timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'create time',
    `mtime`   timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'last updated time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

CREATE TABLE `client_stat`
(
    `client_id` VARCHAR(128) NOT NULL comment 'client id',
    `target`    VARCHAR(100) NOT NULL comment 'target stat platform',
    `port`      int(11)      NOT NULL comment 'client port to get stat information',
    `protocol`  VARCHAR(100) NOT NULL comment 'stat info transport protocol',
    `path`      VARCHAR(128) NOT NULL comment 'stat metric path',
    PRIMARY KEY (`client_id`, `target`, `port`)
) ENGINE = InnoDB;