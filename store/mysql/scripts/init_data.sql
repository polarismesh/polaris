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
SET
    SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";

SET
    time_zone = "+00:00";

USE `polaris_server`;

-- Create a default master account, password is Polarismesh @ 2021
INSERT INTO
    `user` (
        `id`,
        `name`,
        `password`,
        `source`,
        `token`,
        `token_enable`,
        `user_type`,
        `comment`,
        `mobile`,
        `email`,
        `owner`
    )
VALUES
    (
        '65e4789a6d5b49669adf1e9e8387549c',
        'polaris',
        '$2a$10$3izWuZtE5SBdAtSZci.gs.iZ2pAn9I8hEqYrC6gwJp1dyjqQnrrum',
        'Polaris',
        'nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=',
        1,
        20,
        'default polaris admin account',
        '12345678910',
        '12345678910',
        ''
    );

-- Permissions policy inserted into Polaris-Admin
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
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        '(用户) polaris的默认策略',
        'READ_WRITE',
        '65e4789a6d5b49669adf1e9e8387549c',
        'default admin',
        1,
        'Polaris',
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        sysdate (),
        sysdate ()
    );

-- Sport rules inserted into Polaris-Admin to access
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
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        1,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        2,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        3,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        4,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        5,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        6,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        7,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        20,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        21,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        22,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        23,
        '*',
        sysdate (),
        sysdate ()
    );

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
        sysdate (),
        sysdate ()
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
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        1,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        2,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        3,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        4,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        5,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        6,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        7,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        20,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        21,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        22,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'bfa04ae1e32a94fbca9ead86e1ecf581',
        23,
        '*',
        sysdate (),
        sysdate ()
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
        sysdate (),
        sysdate ()
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
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        1,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        2,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        3,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        4,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        5,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        6,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        7,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        20,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        21,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        22,
        '*',
        sysdate (),
        sysdate ()
    ),
    (
        'e3d86e1ecf5812bfa04ae1a94fbca9ea',
        23,
        '*',
        sysdate (),
        sysdate ()
    );

INSERT INTO
    auth_strategy_function (`strategy_id`, `function`)
VALUES
    ('e3d86e1ecf5812bfa04ae1a94fbca9ea', '*');