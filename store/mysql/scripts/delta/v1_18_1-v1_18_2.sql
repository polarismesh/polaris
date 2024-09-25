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