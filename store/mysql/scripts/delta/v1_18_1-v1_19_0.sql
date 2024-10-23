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
/* 角色数据 */
CREATE TABLE
    `auth_role` (
        `id` VARCHAR(128) NOT NULL COMMENT 'role id',
        `name` VARCHAR(100) NOT NULL COMMENT 'role name',
        `owner` VARCHAR(128) NOT NULL COMMENT 'Main account ID',
        `source` VARCHAR(32) NOT NULL COMMENT 'role source',
        `role_type` INT NOT NULL DEFAULT 20 COMMENT 'role type',
        `comment` VARCHAR(255) NOT NULL COMMENT 'describe',
        `flag` TINYINT (4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
        `ctime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
        `mtime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
        `metadata` TEXT COMMENT 'user metadata',
        PRIMARY KEY (`id`),
        UNIQUE KEY (`name`, `owner`),
        KEY `owner` (`owner`),
        KEY `mtime` (`mtime`)
    ) ENGINE = InnoDB;

/* 角色关联用户/用户组关系表 */
CREATE TABLE
    `auth_role_principal` (
        `role_id` VARCHAR(128) NOT NULL COMMENT 'role id',
        `principal_id` VARCHAR(128) NOT NULL COMMENT 'principal id',
        `principal_role` INT NOT NULL COMMENT 'PRINCIPAL type, 1 is User, 2 is Group',
        PRIMARY KEY (`role_id`, `principal_id`, `principal_role`)
    ) ENGINE = InnoDB;

/* 鉴权策略中的资源标签关联信息 */
CRAETE TABLE `auth_strategy_label` (
    `strategy_id` VARCHAR(128) NOT NULL COMMENT 'strategy id',
    `key` VARCHAR(128) NOT NULL COMMENT 'tag key',
    `value` TEXT NOT NULL COMMENT 'tag value',
    `compare_type` VARCHAR(128) NOT NULL COMMENT 'tag kv compare func',
    PRIMARY KEY (`strategy_id`, `key`)
) ENGINE = InnoDB;

/* 鉴权策略中的资源标签关联信息 */
CRAETE TABLE `auth_strategy_function` (
    `strategy_id` VARCHAR(128) NOT NULL COMMENT 'strategy id',
    `function` VARCHAR(256) NOT NULL COMMENT 'server provider function name',
    PRIMARY KEY (`strategy_id`, `function`)
) ENGINE = InnoDB;

/* 默认全局读写以及全局只读策略 */
-- Insert permission policies and association relationships for Polaris-Admin accounts
INSERT INTO
    auth_principal (`strategy_id`, `principal_id`, `principal_role`)
VALUES
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        '65e4789a6d5b49669adf1e9e8387549c',
        1
    );

INSERT INTO
    auth_strategy_function (`strategy_id`, `function`)
VALUES
    ('fbca9bfa04ae4ead86e1ecf5811e32a9', '*');

/* 默认的全局只读策略 */
INSERT INTO
    `auth_strategy` (
        `id`,
        `name`,
        `action`,
        `owner`,
        `comment`,
        `default`,
        `source`,
        `revision`,
        `flag`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        '全局只读策略',
        'ALLOW',
        '65e4789a6d5b49669adf1e9e8387549c',
        'global resources read only',
        1,
        'Polaris',
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        sysdate(),
        sysdate()
    );

INSERT INTO
    `auth_strategy_resource` (
        `strategy_id`,
        `res_type`,
        `res_id`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        0,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        1,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        2,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        3,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        4,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        5,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        6,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        7,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        20,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        21,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        22,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        23,
        '*',
        sysdate(),
        sysdate()
    );

INSERT INTO
    auth_strategy_function (`strategy_id`, `function`)
VALUES
    ('bfa04ae1e32a94fbca9ead86e1ecf581', 'Describe*'),
    ('bfa04ae1e32a94fbca9ead86e1ecf581', 'List*'),
    ('bfa04ae1e32a94fbca9ead86e1ecf581', 'Get*');

/* 默认的全局读写策略 */
INSERT INTO
    `auth_strategy` (
        `id`,
        `name`,
        `action`,
        `owner`,
        `comment`,
        `default`,
        `source`,
        `revision`,
        `flag`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        '全局读写策略',
        'ALLOW',
        '65e4789a6d5b49669adf1e9e8387549c',
        'global resources read and write',
        1,
        'Polaris',
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        sysdate(),
        sysdate()
    );

INSERT INTO
    `auth_strategy_resource` (
        `strategy_id`,
        `res_type`,
        `res_id`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        0,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        1,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        2,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        3,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        4,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        5,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        6,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        7,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        20,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        21,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        22,
        '*',
        sysdate(),
        sysdate()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        23,
        '*',
        sysdate(),
        sysdate()
    );

INSERT INTO
    auth_strategy_function (`strategy_id`, `function`)
VALUES
    ('e3d86e1ecf5812bfa04ae1a94fbca9ea', '*');