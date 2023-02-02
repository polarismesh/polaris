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

CREATE TABLE `leader_election`
(
    `elect_key` VARCHAR(128) NOT NULL,
    `version`   BIGINT NOT NULL DEFAULT 0,
    `leader`    VARCHAR(128) NOT NULL,
    `ctime`     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`elect_key`),
	KEY `version` (`version`)
) engine = innodb;

-- v1.14.0
CREATE TABLE `circuitbreaker_rule_v2`
(
    `id`                VARCHAR(128) NOT NULL,
    `name`              VARCHAR(64)  NOT NULL,
    `namespace`         VARCHAR(64)  NOT NULL default '',
    `enable`            INT          NOT NULL DEFAULT 0,
    `revision`          VARCHAR(40)  NOT NULL,
    `description`       VARCHAR(1024) NOT NULL DEFAULT '',
    `level`             INT          NOT NULL,
    `src_service`        VARCHAR(128) NOT NULL,
    `src_namespace`      VARCHAR(64)  NOT NULL,
    `dst_service`        VARCHAR(128) NOT NULL,
    `dst_namespace`      VARCHAR(64)  NOT NULL,
    `dst_method`         VARCHAR(128) NOT NULL,
    `config`            TEXT,
    `flag`              TINYINT(4)   NOT NULL DEFAULT '0',
    `ctime`             TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`             TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `etime`             TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `mtime` (`mtime`)
) engine = innodb;

CREATE TABLE `fault_detect_rule`
(
    `id`            VARCHAR(128) NOT NULL,
    `name`          VARCHAR(64)  NOT NULL,
    `namespace`     VARCHAR(64)  NOT NULL default 'default',
    `revision`      VARCHAR(40)  NOT NULL,
    `description`   VARCHAR(1024) NOT NULL DEFAULT '',
    `dst_service`    VARCHAR(128) NOT NULL,
    `dst_namespace`  VARCHAR(64)  NOT NULL,
    `dst_method`     VARCHAR(128) NOT NULL,
    `config`        TEXT,
    `flag`          TINYINT(4)   NOT NULL DEFAULT '0',
    `ctime`         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `mtime` (`mtime`)
) engine = innodb;