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
    `id`       VARCHAR(128) PRIMARY KEY,
    `name`     VARCHAR(64) NOT NULL,
    `namespace`     VARCHAR(64) NOT NULL,
    `policy`   VARCHAR(64) NOT NULL,
    `config`   TEXT,
    `enable`   INT         NOT NULL DEFAULT 0,
    `revision` VARCHAR(40) NOT NULL,
    `priority`   smallint(6)    NOT NULL DEFAULT '0' comment 'ratelimit rule priority',
    `flag`     TINYINT(4)  NOT NULL DEFAULT '0',
    `ctime`    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP onupdate CURRENT_TIMESTAMP,
    `etime`    timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `extend_info` VARCHAR(1024) DEFAULT '',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) engine = innodb;


INSERT INTO `service` (`id`,
                       `name`,
                       `namespace`,
                       `comment`,
                       `business`,
                       `token`,
                       `revision`,
                       `owner`,
                       `flag`,
                       `ctime`,
                       `mtime`)
VALUES ('1866010b40be6542db1a2cc846c7f51f',
        'polaris.discover',
        'Polaris',
        'polaris discover service',
        'polaris',
        '2a54df30a6fd4910bdb601dd40b6d58e',
        '5060b13df17240d8-84001e5ae0216c48',
        'polaris',
        0,
        '2021-09-06 07:55:07',
        '2021-09-06 07:55:11'),
       ('846c1866010b40b7f51fe6542db1a2cc',
        'polaris.healthcheck',
        'Polaris',
        'polaris health check service',
        'polaris',
        '254b202a965541a5966b725ae18a6613',
        'aaa44f501ebb4884b0f5c005666ecca1',
        'polaris',
        0,
        '2021-09-06 07:55:07',
        '2021-09-06 07:55:11');
