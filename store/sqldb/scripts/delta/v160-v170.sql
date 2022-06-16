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
VALUES ('e6542db1a2cc846c1866010b40b7f51f',
        'polaris.config',
        'Polaris',
        'polaris config service',
        'polaris',
        'c874d9a0a4b45c82c93e6bf285518c7b',
        '769ec01f58875088faf2cb9e44a4b2d2',
        'polaris',
        0,
        '2021-09-06 07:55:07',
        '2021-09-06 07:55:11');

-- Support polarismesh Authentication Ability
-- User data
CREATE TABLE `user`
(
    `id`           VARCHAR(128) NOT NULL comment 'User ID',
    `name`         VARCHAR(100) NOT NULL comment 'user name',
    `password`     VARCHAR(100) NOT NULL comment 'user password',
    `owner`        VARCHAR(128) NOT NULL comment 'Main account ID',
    `source`       VARCHAR(32)  NOT NULL comment 'Account source',
    `mobile`       VARCHAR(12)  NOT NULL DEFAULT '' comment 'Account mobile phone number',
    `email`        VARCHAR(64)  NOT NULL DEFAULT '' comment 'Account mailbox',
    `token`        VARCHAR(255) NOT NULL comment 'The token information owned by the account can be used for SDK access authentication',
    `token_enable` tinyint(4)   NOT NULL DEFAULT 1,
    `user_type`    int          NOT NULL DEFAULT 20 comment 'Account type, 0 is the admin super account, 20 is the primary account, 50 for the child account',
    `comment`      VARCHAR(255) NOT NULL comment 'describe',
    `flag`         tinyint(4)   NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime`        timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime`        timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- user group
CREATE TABLE `user_group`
(
    `id`           VARCHAR(128) NOT NULL comment 'User group ID',
    `name`         VARCHAR(100) NOT NULL comment 'User group name',
    `owner`        VARCHAR(128) NOT NULL comment 'The main account ID of the user group',
    `token`        VARCHAR(255) NOT NULL comment 'TOKEN information of this user group',
    `comment`      VARCHAR(255) NOT NULL comment 'Description',
    `token_enable` tinyint(4)   NOT NULL DEFAULT 1,
    `flag`         tinyint(4)   NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime`        timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime`        timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- Users of users and user groups
CREATE TABLE `user_group_relation`
(
    `user_id`  VARCHAR(128) NOT NULL comment 'User ID',
    `group_id` VARCHAR(128) NOT NULL comment 'User group ID',
    `ctime`    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime`    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`user_id`, `group_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- Subject information of authentication strategy
CREATE TABLE `auth_strategy`
(
    `id`       VARCHAR(128) NOT NULL comment 'Strategy ID',
    `name`     VARCHAR(100) NOT NULL comment 'Policy name',
    `action`   VARCHAR(32)  NOT NULL comment 'Read and write permission for this policy, only_read = 0, read_write = 1',
    `owner`    VARCHAR(128) NOT NULL comment 'The account ID to which this policy is',
    `comment`  VARCHAR(255) NOT NULL comment 'describe',
    `default`  tinyint(4)   NOT NULL DEFAULT '0',
    `revision` VARCHAR(128) NOT NULL comment 'Authentication rule version',
    `flag`     tinyint(4)   NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime`    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime`    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- Member information, users and user groups in authentication strategies
CREATE TABLE `auth_principal`
(
    `strategy_id`    VARCHAR(128) NOT NULL comment 'Strategy ID',
    `principal_id`   VARCHAR(128) NOT NULL comment 'Principal ID',
    `principal_role` int          NOT NULL comment 'PRINCIPAL type, 1 is User, 2 is Group',
    PRIMARY KEY (`strategy_id`, `principal_id`, `principal_role`)
) ENGINE = InnoDB;

-- Resource information record involved in the authentication strategy
CREATE TABLE `auth_strategy_resource`
(
    `strategy_id` VARCHAR(128) NOT NULL comment 'Strategy ID',
    `res_type`    int          NOT NULL comment 'Resource Type, Namespaces = 0, Service = 1, configgroups = 2',
    `res_id`      VARCHAR(128) NOT NULL comment 'Resource ID',
    `ctime`       timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime`       timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`strategy_id`, `res_type`, `res_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- Create a default master account, password is Polarismesh @ 2021
INSERT INTO `user` (`id`,
                    `name`,
                    `password`,
                    `source`,
                    `token`,
                    `token_enable`,
                    `user_type`,
                    `comment`,
                    `owner`)
VALUES ('65e4789a6d5b49669adf1e9e8387549c',
        'polaris',
        '$2a$10$3izWuZtE5SBdAtSZci.gs.iZ2pAn9I8hEqYrC6gwJp1dyjqQnrrum',
        'Polaris',
        'nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=',
        1,
        20,
        'default polaris admin account',
        '');

-- Permissions policy inserted into Polaris-Admin
INSERT INTO `auth_strategy`(`id`,
                            `name`,
                            `action`,
                            `owner`,
                            `comment`,
                            `default`,
                            `revision`,
                            `flag`,
                            `ctime`,
                            `mtime`)
VALUES ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        '(用户) polaris的默认策略',
        'READ_WRITE',
        '65e4789a6d5b49669adf1e9e8387549c',
        'default admin',
        1,
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        sysdate(),
        sysdate());

-- Sport rules inserted into Polaris-Admin to access
INSERT INTO `auth_strategy_resource`(`strategy_id`,
                                     `res_type`,
                                     `res_id`,
                                     `ctime`,
                                     `mtime`)
VALUES ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        '*',
        sysdate(),
        sysdate()),
       ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        1,
        '*',
        sysdate(),
        sysdate()),
       ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        2,
        '*',
        sysdate(),
        sysdate());

-- Insert permission policies and association relationships for Polaris-Admin accounts
INSERT INTO auth_principal(`strategy_id`, `principal_id`, `principal_role`) VALUE (
                                                                                   'fbca9bfa04ae4ead86e1ecf5811e32a9',
                                                                                   '65e4789a6d5b49669adf1e9e8387549c',
                                                                                   1
    );