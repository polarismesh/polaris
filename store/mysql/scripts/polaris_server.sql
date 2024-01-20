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
SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";

SET time_zone = "+00:00";

--
-- Database: `polaris_server`
--
CREATE DATABASE IF NOT EXISTS `polaris_server` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;

USE `polaris_server`;

-- --------------------------------------------------------
--
-- Table structure `instance`
--
CREATE TABLE `instance`
(
    `id`                  VARCHAR(128) NOT NULL COMMENT 'Unique ID',
    `service_id`          VARCHAR(32)  NOT NULL COMMENT 'Service ID',
    `vpc_id`              VARCHAR(64)           DEFAULT NULL COMMENT 'VPC ID',
    `host`                VARCHAR(128) NOT NULL COMMENT 'instance Host Information',
    `port`                INT(11)      NOT NULL COMMENT 'instance port information',
    `protocol`            VARCHAR(32)           DEFAULT NULL COMMENT 'Listening protocols for corresponding ports, such as TPC, UDP, GRPC, DUBBO, etc.',
    `version`             VARCHAR(32)           DEFAULT NULL COMMENT 'The version of the instance can be used for version routing',
    `health_status`       TINYINT(4)   NOT NULL DEFAULT '1' COMMENT 'The health status of the instance, 1 is health, 0 is unhealthy',
    `isolate`             TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Example isolation status flag, 0 is not isolated, 1 is isolated',
    `weight`              SMALLINT(6)  NOT NULL DEFAULT '100' COMMENT 'The weight of the instance is mainly used for LoadBalance, default is 100',
    `enable_health_check` TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Whether to open a heartbeat on an instance, check the logic, 0 is not open, 1 is open',
    `logic_set`           VARCHAR(128)          DEFAULT NULL COMMENT 'Example logic packet information',
    `cmdb_region`         VARCHAR(128)          DEFAULT NULL COMMENT 'The region information of the instance is mainly used to close the route',
    `cmdb_zone`           VARCHAR(128)          DEFAULT NULL COMMENT 'The ZONE information of the instance is mainly used to close the route.',
    `cmdb_idc`            VARCHAR(128)          DEFAULT NULL COMMENT 'The IDC information of the instance is mainly used to close the route',
    `priority`            TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Example priority, currently useless',
    `revision`            VARCHAR(32)  NOT NULL COMMENT 'Instance version information',
    `flag`                TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime`               TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`               TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `service_id` (`service_id`),
    KEY `mtime` (`mtime`),
    KEY `host` (`host`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `health_check`
--
CREATE TABLE `health_check`
(
    `id`   VARCHAR(128) NOT NULL COMMENT 'Instance ID',
    `type` TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Instance health check type',
    `ttl`  INT(11)      NOT NULL COMMENT 'TTL time jumping',
    PRIMARY KEY (`id`)
    /* CONSTRAINT `health_check_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE */
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `instance_metadata`
--
CREATE TABLE `instance_metadata`
(
    `id`     VARCHAR(128)  NOT NULL COMMENT 'Instance ID',
    `mkey`   VARCHAR(128)  NOT NULL COMMENT 'instance label of Key',
    `mvalue` VARCHAR(4096) NOT NULL COMMENT 'instance label Value',
    `ctime`  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`, `mkey`),
    KEY `mkey` (`mkey`)
    /* CONSTRAINT `instance_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE */
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `namespace`
--
CREATE TABLE `namespace`
(
    `name`    VARCHAR(64)   NOT NULL COMMENT 'Namespace name, unique',
    `comment` VARCHAR(1024)          DEFAULT NULL COMMENT 'Description of namespace',
    `token`   VARCHAR(64)   NOT NULL COMMENT 'TOKEN named space for write operation check',
    `owner`   VARCHAR(1024) NOT NULL COMMENT 'Responsible for named space Owner',
    `flag`    TINYINT(4)    NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime`   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    `service_export_to` TEXT COMMENT 'namespace metadata',
    `metadata` TEXT COMMENT 'namespace metadata',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB;

--
-- Data in the conveyor `namespace`
--
INSERT INTO `namespace` (`name`,
                         `comment`,
                         `token`,
                         `owner`,
                         `flag`,
                         `ctime`,
                         `mtime`)
VALUES ('Polaris',
        'Polaris-server',
        '2d1bfe5d12e04d54b8ee69e62494c7fd',
        'polaris',
        0,
        '2019-09-06 07:55:07',
        '2019-09-06 07:55:07'),
       ('default',
        'Default Environment',
        'e2e473081d3d4306b52264e49f7ce227',
        'polaris',
        0,
        '2021-07-27 19:37:37',
        '2021-07-27 19:37:37');

-- --------------------------------------------------------
--
-- Table structure `routing_config`
--
CREATE TABLE `routing_config`
(
    `id`         VARCHAR(32) NOT NULL COMMENT 'Routing configuration ID',
    `in_bounds`  TEXT COMMENT 'Service is routing rules',
    `out_bounds` TEXT COMMENT 'Service main routing rules',
    `revision`   VARCHAR(40) NOT NULL COMMENT 'Routing rule version',
    `flag`       TINYINT(4)  NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime`      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `ratelimit_config`
--
CREATE TABLE `ratelimit_config`
(
    `id`         VARCHAR(32)  NOT NULL COMMENT 'ratelimit rule ID',
    `name`       VARCHAR(64)  NOT NULL COMMENT 'ratelimt rule name',
    `disable`    TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'ratelimit disable',
    `service_id` VARCHAR(32)  NOT NULL COMMENT 'Service ID',
    `method`     VARCHAR(512) NOT NULL COMMENT 'ratelimit method',
    `labels`     TEXT         NOT NULL COMMENT 'Conductive flow for a specific label',
    `priority`   SMALLINT(6)  NOT NULL DEFAULT '0' COMMENT 'ratelimit rule priority',
    `rule`       TEXT         NOT NULL COMMENT 'Current limiting rules',
    `revision`   VARCHAR(32)  NOT NULL COMMENT 'Limiting version',
    `flag`       TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime`      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    `etime`      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'RateLimit rule enable time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`),
    KEY `service_id` (`service_id`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `ratelimit_revision`
--
CREATE TABLE `ratelimit_revision`
(
    `service_id`    VARCHAR(32) NOT NULL COMMENT 'Service ID',
    `last_revision` VARCHAR(40) NOT NULL COMMENT 'The latest limited limiting rule version of the corresponding service',
    `mtime`         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`service_id`),
    KEY `service_id` (`service_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `service`
--
CREATE TABLE `service`
(
    `id`           VARCHAR(32)   NOT NULL COMMENT 'Service ID',
    `name`         VARCHAR(128)  NOT NULL COMMENT 'Service name, only under the namespace',
    `namespace`    VARCHAR(64)   NOT NULL COMMENT 'Namespace belongs to the service',
    `ports`        TEXT                   DEFAULT NULL COMMENT 'Service will have a list of all port information of the external exposure (single process exposing multiple protocols)',
    `business`     VARCHAR(64)            DEFAULT NULL COMMENT 'Service business information',
    `department`   VARCHAR(1024)          DEFAULT NULL COMMENT 'Service department information',
    `cmdb_mod1`    VARCHAR(1024)          DEFAULT NULL COMMENT '',
    `cmdb_mod2`    VARCHAR(1024)          DEFAULT NULL COMMENT '',
    `cmdb_mod3`    VARCHAR(1024)          DEFAULT NULL COMMENT '',
    `comment`      VARCHAR(1024)          DEFAULT NULL COMMENT 'Description information',
    `token`        VARCHAR(2048) NOT NULL COMMENT 'Service token, used to handle all the services involved in the service',
    `revision`     VARCHAR(32)   NOT NULL COMMENT 'Service version information',
    `owner`        VARCHAR(1024) NOT NULL COMMENT 'Owner information belonging to the service',
    `flag`         TINYINT(4)    NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `reference`    VARCHAR(32)            DEFAULT NULL COMMENT 'Service alias, what is the actual service name that the service is actually pointed out?',
    `refer_filter` VARCHAR(1024)          DEFAULT NULL COMMENT '',
    `platform_id`  VARCHAR(32)            DEFAULT '' COMMENT 'The platform ID to which the service belongs',
    `ctime`        TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`        TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    `export_to` TEXT COMMENT 'service export to some namespace',
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`, `namespace`),
    KEY `namespace` (`namespace`),
    KEY `mtime` (`mtime`),
    KEY `reference` (`reference`),
    KEY `platform_id` (`platform_id`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Data in the conveyor `service`
--
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
VALUES ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        'polaris.checker',
        'Polaris',
        'polaris checker service',
        'polaris',
        '7d19c46de327408d8709ee7392b7700b',
        '301b1e9f0bbd47a6b697e26e99dfe012',
        'polaris',
        0,
        '2021-09-06 07:55:07',
        '2021-09-06 07:55:09');

-- --------------------------------------------------------
--
-- Table structure `service_metadata`
--
CREATE TABLE `service_metadata`
(
    `id`     VARCHAR(32)   NOT NULL COMMENT 'Service ID',
    `mkey`   VARCHAR(128)  NOT NULL COMMENT 'Service label key',
    `mvalue` VARCHAR(4096) NOT NULL COMMENT 'Service label Value',
    `ctime`  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`, `mkey`),
    KEY `mkey` (`mkey`)
    /* CONSTRAINT `service_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE */
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `owner_service_map`Quickly query all services under an Owner
--
CREATE TABLE
    `owner_service_map`
(
    `id`        VARCHAR(32)  NOT NULL COMMENT '',
    `owner`     VARCHAR(32)  NOT NULL COMMENT 'Service Owner',
    `service`   VARCHAR(128) NOT NULL COMMENT 'service name',
    `namespace` VARCHAR(64)  NOT NULL COMMENT 'namespace name',
    PRIMARY KEY (`id`),
    KEY `owner` (`owner`),
    KEY `name` (`service`, `namespace`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `circuitbreaker_rule`
--
CREATE TABLE `circuitbreaker_rule`
(
    `id`         VARCHAR(97)   NOT NULL COMMENT 'Melting rule ID',
    `version`    VARCHAR(32)   NOT NULL DEFAULT 'master' COMMENT 'Melting rule version, default is MASTR',
    `name`       VARCHAR(128)  NOT NULL COMMENT 'Melting rule name',
    `namespace`  VARCHAR(64)   NOT NULL COMMENT 'Melting rule belongs to name space',
    `business`   VARCHAR(64)            DEFAULT NULL COMMENT 'Business information of fuse regular',
    `department` VARCHAR(1024)          DEFAULT NULL COMMENT 'Department information to which the fuse regular belongs',
    `comment`    VARCHAR(1024)          DEFAULT NULL COMMENT 'Description of the fuse rule',
    `inbounds`   TEXT          NOT NULL COMMENT 'Service-tuned fuse rule',
    `outbounds`  TEXT          NOT NULL COMMENT 'Service Motoring Fuse Rule',
    `token`      VARCHAR(32)   NOT NULL COMMENT 'Token, which is fucking, mainly for writing operation check',
    `owner`      VARCHAR(1024) NOT NULL COMMENT 'Melting rule Owner information',
    `revision`   VARCHAR(32)   NOT NULL COMMENT 'Melt rule version information',
    `flag`       TINYINT(4)    NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime`      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`, `version`),
    UNIQUE KEY `name` (`name`, `namespace`, `version`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `circuitbreaker_rule_relation`
--
CREATE TABLE `circuitbreaker_rule_relation`
(
    `service_id`   VARCHAR(32) NOT NULL COMMENT 'Service ID',
    `rule_id`      VARCHAR(97) NOT NULL COMMENT 'Melting rule ID',
    `rule_version` VARCHAR(32) NOT NULL COMMENT 'Melting rule version',
    `flag`         TINYINT(4)  NOT NULL DEFAULT '0' COMMENT 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime`        TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`        TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`service_id`),
    KEY `mtime` (`mtime`),
    KEY `rule_id` (`rule_id`)
    /* CONSTRAINT `circuitbreaker_rule_relation_ibfk_1` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE */
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `t_ip_config`
--
CREATE TABLE `t_ip_config`
(
    `Fip`     INT(10) UNSIGNED NOT NULL COMMENT 'Machine IP',
    `FareaId` INT(10) UNSIGNED NOT NULL COMMENT 'Area number',
    `FcityId` INT(10) UNSIGNED NOT NULL COMMENT 'City number',
    `FidcId`  INT(10) UNSIGNED NOT NULL COMMENT 'IDC number',
    `Fflag`   TINYINT(4) DEFAULT '0',
    `Fstamp`  DATETIME         NOT NULL,
    `Fflow`   INT(10) UNSIGNED NOT NULL,
    PRIMARY KEY (`Fip`),
    KEY `idx_Fflow` (`Fflow`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `t_policy`
--
CREATE TABLE `t_policy`
(
    `FmodId` INT(10) UNSIGNED NOT NULL,
    `Fdiv`   INT(10) UNSIGNED NOT NULL,
    `Fmod`   INT(10) UNSIGNED NOT NULL,
    `Fflag`  TINYINT(4) DEFAULT '0',
    `Fstamp` DATETIME         NOT NULL,
    `Fflow`  INT(10) UNSIGNED NOT NULL,
    PRIMARY KEY (`FmodId`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `t_route`
--
CREATE TABLE `t_route`
(
    `Fip`    INT(10) UNSIGNED NOT NULL,
    `FmodId` INT(10) UNSIGNED NOT NULL,
    `FcmdId` INT(10) UNSIGNED NOT NULL,
    `FsetId` VARCHAR(32)      NOT NULL,
    `Fflag`  TINYINT(4) DEFAULT '0',
    `Fstamp` DATETIME         NOT NULL,
    `Fflow`  INT(10) UNSIGNED NOT NULL,
    PRIMARY KEY (`Fip`, `FmodId`, `FcmdId`),
    KEY `Fflow` (`Fflow`),
    KEY `idx1` (`FmodId`, `FcmdId`, `FsetId`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `t_section`
--
CREATE TABLE `t_section`
(
    `FmodId` INT(10) UNSIGNED NOT NULL,
    `Ffrom`  INT(10) UNSIGNED NOT NULL,
    `Fto`    INT(10) UNSIGNED NOT NULL,
    `Fxid`   INT(10) UNSIGNED NOT NULL,
    `Fflag`  TINYINT(4) DEFAULT '0',
    `Fstamp` DATETIME         NOT NULL,
    `Fflow`  INT(10) UNSIGNED NOT NULL,
    PRIMARY KEY (`FmodId`, `Ffrom`, `Fto`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure `start_lock`
--
CREATE TABLE `start_lock`
(
    `lock_id`  INT(11)     NOT NULL COMMENT '锁序号',
    `lock_key` VARCHAR(32) NOT NULL COMMENT 'Lock name',
    `server`   VARCHAR(32) NOT NULL COMMENT 'SERVER holding launch lock',
    `mtime`    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    PRIMARY KEY (`lock_id`, `lock_key`)
) ENGINE = InnoDB;

--
-- Data in the conveyor `start_lock`
--
INSERT INTO `start_lock` (`lock_id`, `lock_key`, `server`, `mtime`)
VALUES (1, 'sz', 'aaa', '2019-12-05 08:35:49');

-- --------------------------------------------------------
--
-- Table structure `cl5_module`
--
CREATE TABLE `cl5_module`
(
    `module_id`    INT(11)   NOT NULL COMMENT 'Module ID',
    `interface_id` INT(11)   NOT NULL COMMENT 'Interface ID',
    `range_num`    INT(11)   NOT NULL,
    `mtime`        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`module_id`)
) ENGINE = InnoDB COMMENT = 'To generate SID';

--
-- Data in the conveyor `cl5_module`
--
INSERT INTO cl5_module (module_id, interface_id, range_num)
VALUES (3000001, 1, 0);

-- --------------------------------------------------------
--
-- Table structure `config_file`
--
CREATE TABLE `config_file`
(
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `namespace`   VARCHAR(64)     NOT NULL COMMENT '所属的namespace',
    `group`       VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '所属的文件组',
    `name`        VARCHAR(128)    NOT NULL COMMENT '配置文件名',
    `content`     LONGTEXT        NOT NULL COMMENT '文件内容',
    `format`      VARCHAR(16)              DEFAULT 'text' COMMENT '文件格式，枚举值',
    `comment`     VARCHAR(512)             DEFAULT NULL COMMENT '备注信息',
    `flag`        TINYINT(4)      NOT NULL DEFAULT '0' COMMENT '软删除标记位',
    `create_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32)              DEFAULT NULL COMMENT '创建人',
    `modify_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32)              DEFAULT NULL COMMENT '最后更新人',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_file` (`namespace`, `group`, `name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1 COMMENT = '配置文件表';

-- --------------------------------------------------------
--
-- Table structure `config_file_group`
--
CREATE TABLE `config_file_group`
(
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name`        VARCHAR(128)    NOT NULL COMMENT '配置文件分组名',
    `namespace`   VARCHAR(64)     NOT NULL COMMENT '所属的namespace',
    `comment`     VARCHAR(512)             DEFAULT NULL COMMENT '备注信息',
    `owner`       VARCHAR(1024)            DEFAULT NULL COMMENT '负责人',
    `create_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32)              DEFAULT NULL COMMENT '创建人',
    `modify_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32)              DEFAULT NULL COMMENT '最后更新人',
    `business`    VARCHAR(64)              DEFAULT NULL COMMENT 'Service business information',
    `department`  VARCHAR(1024)            DEFAULT NULL COMMENT 'Service department information',
    `metadata`    TEXT COMMENT '配置分组标签',
    `flag`        TINYINT(4)      NOT NULL DEFAULT '0' COMMENT '是否被删除',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`namespace`, `name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1 COMMENT = '配置文件组表';

-- --------------------------------------------------------
--
-- Table structure `config_file_release`
--
CREATE TABLE `config_file_release`
(
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name`         VARCHAR(128)             DEFAULT NULL COMMENT '发布标题',
    `namespace`    VARCHAR(64)     NOT NULL COMMENT '所属的namespace',
    `group`        VARCHAR(128)    NOT NULL COMMENT '所属的文件组',
    `file_name`    VARCHAR(128)    NOT NULL COMMENT '配置文件名',
    `format`       VARCHAR(16)              DEFAULT 'text' COMMENT '文件格式，枚举值',
    `content`      LONGTEXT        NOT NULL COMMENT '文件内容',
    `comment`      VARCHAR(512)             DEFAULT NULL COMMENT '备注信息',
    `md5`          VARCHAR(128)    NOT NULL COMMENT 'content的md5值',
    `version`      BIGINT(11)      NOT NULL COMMENT '版本号，每次发布自增1',
    `flag`         TINYINT(4)      NOT NULL DEFAULT '0' COMMENT '是否被删除',
    `create_time`  TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`    VARCHAR(32)              DEFAULT NULL COMMENT '创建人',
    `modify_time`  TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`    VARCHAR(32)              DEFAULT NULL COMMENT '最后更新人',
    `tags`         TEXT COMMENT '文件标签',
    `active`       TINYINT(4)      NOT NULL DEFAULT '0' COMMENT '是否处于使用中',
    `description`  VARCHAR(512)             DEFAULT NULL COMMENT '发布描述',
    `release_type` VARCHAR(25)     NOT NULL DEFAULT '' COMMENT '文件类型：""：全量 gray：灰度',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_file` (`namespace`, `group`, `file_name`, `name`),
    KEY `idx_modify_time` (`modify_time`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1 COMMENT = '配置文件发布表';


-- --------------------------------------------------------
--
-- Table structure `config_file_release_history`
--
CREATE TABLE `config_file_release_history`
(
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name`        VARCHAR(64)              DEFAULT '' COMMENT '发布名称',
    `namespace`   VARCHAR(64)     NOT NULL COMMENT '所属的namespace',
    `group`       VARCHAR(128)    NOT NULL COMMENT '所属的文件组',
    `file_name`   VARCHAR(128)    NOT NULL COMMENT '配置文件名',
    `content`     LONGTEXT        NOT NULL COMMENT '文件内容',
    `format`      VARCHAR(16)              DEFAULT 'text' COMMENT '文件格式',
    `comment`     VARCHAR(512)             DEFAULT NULL COMMENT '备注信息',
    `md5`         VARCHAR(128)    NOT NULL COMMENT 'content的md5值',
    `type`        VARCHAR(32)     NOT NULL COMMENT '发布类型，例如全量发布、灰度发布',
    `status`      VARCHAR(16)     NOT NULL DEFAULT 'success' COMMENT '发布状态，success表示成功，fail 表示失败',
    `create_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32)              DEFAULT NULL COMMENT '创建人',
    `modify_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32)              DEFAULT NULL COMMENT '最后更新人',
    `tags`        TEXT COMMENT '文件标签',
    `version`     BIGINT(11) COMMENT '版本号，每次发布自增1',
    `reason`      VARCHAR(3000)            DEFAULT '' COMMENT '原因',
    `description` VARCHAR(512)             DEFAULT NULL COMMENT '发布描述',
    PRIMARY KEY (`id`),
    KEY `idx_file` (`namespace`, `group`, `file_name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1 COMMENT = '配置文件发布历史表';

-- --------------------------------------------------------
--
-- Table structure `config_file_tag`
--
CREATE TABLE `config_file_tag`
(
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `key`         VARCHAR(128)    NOT NULL COMMENT 'tag 的键',
    `Value`       VARCHAR(128)    NOT NULL COMMENT 'tag 的值',
    `namespace`   VARCHAR(64)     NOT NULL COMMENT '所属的namespace',
    `group`       VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '所属的文件组',
    `file_name`   VARCHAR(128)    NOT NULL COMMENT '配置文件名',
    `create_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32)              DEFAULT NULL COMMENT '创建人',
    `modify_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32)              DEFAULT NULL COMMENT '最后更新人',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tag` (`key`, `Value`, `namespace`, `group`, `file_name`),
    KEY `idx_file` (`namespace`, `group`, `file_name`)
) ENGINE = InnoDB COMMENT = '配置文件标签表';

CREATE TABLE `user`
(
    `id`           VARCHAR(128) NOT NULL COMMENT 'User ID',
    `name`         VARCHAR(100) NOT NULL COMMENT 'user name',
    `password`     VARCHAR(100) NOT NULL COMMENT 'user password',
    `owner`        VARCHAR(128) NOT NULL COMMENT 'Main account ID',
    `source`       VARCHAR(32)  NOT NULL COMMENT 'Account source',
    `mobile`       VARCHAR(12)  NOT NULL DEFAULT '' COMMENT 'Account mobile phone number',
    `email`        VARCHAR(64)  NOT NULL DEFAULT '' COMMENT 'Account mailbox',
    `token`        VARCHAR(255) NOT NULL COMMENT 'The token information owned by the account can be used for SDK access authentication',
    `token_enable` TINYINT(4)   NOT NULL DEFAULT 1,
    `user_type`    INT          NOT NULL DEFAULT 20 COMMENT 'Account type, 0 is the admin super account, 20 is the primary account, 50 for the child account',
    `comment`      VARCHAR(255) NOT NULL COMMENT 'describe',
    `flag`         TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

CREATE TABLE `user_group`
(
    `id`           VARCHAR(128) NOT NULL COMMENT 'User group ID',
    `name`         VARCHAR(100) NOT NULL COMMENT 'User group name',
    `owner`        VARCHAR(128) NOT NULL COMMENT 'The main account ID of the user group',
    `token`        VARCHAR(255) NOT NULL COMMENT 'TOKEN information of this user group',
    `comment`      VARCHAR(255) NOT NULL COMMENT 'Description',
    `token_enable` TINYINT(4)   NOT NULL DEFAULT 1,
    `flag`         TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

CREATE TABLE `user_group_relation`
(
    `user_id`  VARCHAR(128) NOT NULL COMMENT 'User ID',
    `group_id` VARCHAR(128) NOT NULL COMMENT 'User group ID',
    `ctime`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`user_id`, `group_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

CREATE TABLE `auth_strategy`
(
    `id`       VARCHAR(128) NOT NULL COMMENT 'Strategy ID',
    `name`     VARCHAR(100) NOT NULL COMMENT 'Policy name',
    `action`   VARCHAR(32)  NOT NULL COMMENT 'Read and write permission for this policy, only_read = 0, read_write = 1',
    `owner`    VARCHAR(128) NOT NULL COMMENT 'The account ID to which this policy is',
    `comment`  VARCHAR(255) NOT NULL COMMENT 'describe',
    `default`  TINYINT(4)   NOT NULL DEFAULT '0',
    `revision` VARCHAR(128) NOT NULL COMMENT 'Authentication rule version',
    `flag`     TINYINT(4)   NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

CREATE TABLE `auth_principal`
(
    `strategy_id`    VARCHAR(128) NOT NULL COMMENT 'Strategy ID',
    `principal_id`   VARCHAR(128) NOT NULL COMMENT 'Principal ID',
    `principal_role` INT          NOT NULL COMMENT 'PRINCIPAL type, 1 is User, 2 is Group',
    PRIMARY KEY (`strategy_id`, `principal_id`, `principal_role`)
) ENGINE = InnoDB;

CREATE TABLE `auth_strategy_resource`
(
    `strategy_id` VARCHAR(128) NOT NULL COMMENT 'Strategy ID',
    `res_type`    INT          NOT NULL COMMENT 'Resource Type, Namespaces = 0, Service = 1, configgroups = 2',
    `res_id`      VARCHAR(128) NOT NULL COMMENT 'Resource ID',
    `ctime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
    `mtime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last updated time',
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
                    `mobile`,
                    `email`,
                    `owner`)
VALUES ('65e4789a6d5b49669adf1e9e8387549c',
        'polaris',
        '$2a$10$3izWuZtE5SBdAtSZci.gs.iZ2pAn9I8hEqYrC6gwJp1dyjqQnrrum',
        'Polaris',
        'nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=',
        1,
        20,
        'default polaris admin account',
        '12345678910',
        '12345678910',
        '');

-- Permissions policy inserted into Polaris-Admin
INSERT INTO `auth_strategy` (`id`,
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
        SYSDATE(),
        SYSDATE());

-- Sport rules inserted into Polaris-Admin to access
INSERT INTO `auth_strategy_resource` (`strategy_id`,
                                      `res_type`,
                                      `res_id`,
                                      `ctime`,
                                      `mtime`)
VALUES ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        '*',
        SYSDATE(),
        SYSDATE()),
       ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        1,
        '*',
        SYSDATE(),
        SYSDATE()),
       ('fbca9bfa04ae4ead86e1ecf5811e32a9',
        2,
        '*',
        SYSDATE(),
        SYSDATE());

-- Insert permission policies and association relationships for Polaris-Admin accounts
INSERT INTO auth_principal (`strategy_id`, `principal_id`, `principal_role`) VALUE (
                                                                                    'fbca9bfa04ae4ead86e1ecf5811e32a9',
                                                                                    '65e4789a6d5b49669adf1e9e8387549c',
                                                                                    1
    );

-- v1.8.0, support client info storage
CREATE TABLE `client`
(
    `id`      VARCHAR(128) NOT NULL COMMENT 'client id',
    `host`    VARCHAR(100) NOT NULL COMMENT 'client host IP',
    `type`    VARCHAR(100) NOT NULL COMMENT 'client type: polaris-java/polaris-go',
    `version` VARCHAR(32)  NOT NULL COMMENT 'client SDK version',
    `region`  VARCHAR(128)          DEFAULT NULL COMMENT 'region info for client',
    `zone`    VARCHAR(128)          DEFAULT NULL COMMENT 'zone info for client',
    `campus`  VARCHAR(128)          DEFAULT NULL COMMENT 'campus info for client',
    `flag`    TINYINT(4)   NOT NULL DEFAULT '0' COMMENT '0 is valid, 1 is invalid(deleted)',
    `ctime`   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
    `mtime`   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'last updated time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB;

CREATE TABLE `client_stat`
(
    `client_id` VARCHAR(128) NOT NULL COMMENT 'client id',
    `target`    VARCHAR(100) NOT NULL COMMENT 'target stat platform',
    `port`      INT(11)      NOT NULL COMMENT 'client port to get stat information',
    `protocol`  VARCHAR(100) NOT NULL COMMENT 'stat info transport protocol',
    `path`      VARCHAR(128) NOT NULL COMMENT 'stat metric path',
    PRIMARY KEY (`client_id`, `target`, `port`)
) ENGINE = InnoDB;

-- v1.9.0
CREATE TABLE `config_file_template`
(
    `id`          BIGINT(10) UNSIGNED           NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name`        VARCHAR(128) COLLATE utf8_bin NOT NULL COMMENT '配置文件模板名称',
    `content`     LONGTEXT COLLATE utf8_bin     NOT NULL COMMENT '配置文件模板内容',
    `format`      VARCHAR(16) COLLATE utf8_bin           DEFAULT 'text' COMMENT '模板文件格式',
    `comment`     VARCHAR(512) COLLATE utf8_bin          DEFAULT NULL COMMENT '模板描述信息',
    `create_time` TIMESTAMP                     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32) COLLATE utf8_bin           DEFAULT NULL COMMENT '创建人',
    `modify_time` TIMESTAMP                     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32) COLLATE utf8_bin           DEFAULT NULL COMMENT '最后更新人',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = utf8
  COLLATE = utf8_bin COMMENT = '配置文件模板表';

INSERT INTO `config_file_template` (`name`,
                                    `content`,
                                    `format`,
                                    `comment`,
                                    `create_time`,
                                    `create_by`,
                                    `modify_time`,
                                    `modify_by`)
VALUES ('spring-cloud-gateway-braining',
        '{
        "rules":[
            {
                "conditions":[
                    {
                        "key":"${http.query.uid}",
                        "values":["10000"],
                        "operation":"EQUALS"
                    }
                ],
                "labels":[
                    {
                        "key":"env",
                        "value":"green"
                    }
                ]
            }
        ]
    }',
        'json',
        'Spring Cloud Gateway  染色规则',
        NOW(),
        'polaris',
        NOW(),
        'polaris');

-- v1.12.0
CREATE TABLE `routing_config_v2`
(
    `id`          VARCHAR(128) NOT NULL,
    `name`        VARCHAR(64)  NOT NULL DEFAULT '',
    `namespace`   VARCHAR(64)  NOT NULL DEFAULT '',
    `policy`      VARCHAR(64)  NOT NULL,
    `config`      TEXT,
    `enable`      INT          NOT NULL DEFAULT 0,
    `revision`    VARCHAR(40)  NOT NULL,
    `description` VARCHAR(500) NOT NULL DEFAULT '',
    `priority`    SMALLINT(6)  NOT NULL DEFAULT '0' COMMENT 'ratelimit rule priority',
    `flag`        TINYINT(4)   NOT NULL DEFAULT '0',
    `ctime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `etime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `extend_info` VARCHAR(1024)         DEFAULT '',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = innodb;

CREATE TABLE
    `leader_election`
(
    `elect_key` VARCHAR(128) NOT NULL,
    `version`   BIGINT       NOT NULL DEFAULT 0,
    `leader`    VARCHAR(128) NOT NULL,
    `ctime`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`elect_key`),
    KEY `version` (`version`)
) ENGINE = innodb;

-- v1.14.0
CREATE TABLE `circuitbreaker_rule_v2`
(
    `id`            VARCHAR(128)  NOT NULL,
    `name`          VARCHAR(64)   NOT NULL,
    `namespace`     VARCHAR(64)   NOT NULL DEFAULT '',
    `enable`        INT           NOT NULL DEFAULT 0,
    `revision`      VARCHAR(40)   NOT NULL,
    `description`   VARCHAR(1024) NOT NULL DEFAULT '',
    `level`         INT           NOT NULL,
    `src_service`   VARCHAR(128)  NOT NULL,
    `src_namespace` VARCHAR(64)   NOT NULL,
    `dst_service`   VARCHAR(128)  NOT NULL,
    `dst_namespace` VARCHAR(64)   NOT NULL,
    `dst_method`    VARCHAR(128)  NOT NULL,
    `config`        TEXT,
    `flag`          TINYINT(4)    NOT NULL DEFAULT '0',
    `ctime`         TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`         TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `etime`         TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `mtime` (`mtime`)
) ENGINE = innodb;

CREATE TABLE `fault_detect_rule`
(
    `id`            VARCHAR(128)  NOT NULL,
    `name`          VARCHAR(64)   NOT NULL,
    `namespace`     VARCHAR(64)   NOT NULL DEFAULT 'default',
    `revision`      VARCHAR(40)   NOT NULL,
    `description`   VARCHAR(1024) NOT NULL DEFAULT '',
    `dst_service`   VARCHAR(128)  NOT NULL,
    `dst_namespace` VARCHAR(64)   NOT NULL,
    `dst_method`    VARCHAR(128)  NOT NULL,
    `config`        TEXT,
    `flag`          TINYINT(4)    NOT NULL DEFAULT '0',
    `ctime`         TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`         TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `mtime` (`mtime`)
) ENGINE = innodb;

/* 服务契约表 */
CREATE TABLE service_contract
(
    `id`        VARCHAR(128) NOT NULL COMMENT '服务契约主键',
    `name`      VARCHAR(128) NOT NULL COMMENT '服务契约名称',
    `namespace` VARCHAR(64)  NOT NULL COMMENT '命名空间',
    `service`   VARCHAR(128) NOT NULL COMMENT '服务名称',
    `protocol`  VARCHAR(32)  NOT NULL COMMENT '当前契约对应的协议信息 e.g. http/dubbo/grpc/thrift',
    `version`   VARCHAR(64)  NOT NULL COMMENT '服务契约版本',
    `revision`  VARCHAR(128) NOT NULL COMMENT '当前服务契约的全部内容版本摘要',
    `flag`      TINYINT(4)            DEFAULT 0 COMMENT '逻辑删除标志位 ， 0 位有效 ， 1 为逻辑删除',
    `content`   LONGTEXT COMMENT '描述信息',
    `ctime`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    -- 通过 服务 + 协议信息 + 契约版本 + 名称 进行一次 hash 计算，作为主键
    PRIMARY KEY (`id`),
    -- 服务 + 协议信息 + 契约版本 + 辅助标签 必须保证唯一
    KEY (
         `namespace`,
         `service`,
         `name`,
         `version`,
         `protocol`
        )
) ENGINE = InnoDB;

/* 服务契约中针对单个接口定义的详细信息描述表 */
CREATE TABLE service_contract_detail
(
    `id`          VARCHAR(128) NOT NULL COMMENT '服务契约单个接口定义记录主键',
    `contract_id` VARCHAR(128) NOT NULL COMMENT '服务契约 ID',
    `method`      VARCHAR(32)  NOT NULL COMMENT 'http协议中的 method 字段, eg:POST/GET/PUT/DELETE, 其他 gRPC 可以用来标识 stream 类型',
    `path`        VARCHAR(128) NOT NULL COMMENT '接口具体全路径描述',
    `source`      INT COMMENT '该条记录来源, 0:SDK/1:MANUAL',
    `content`     LONGTEXT COMMENT '描述信息',
    `revision`    VARCHAR(128) NOT NULL COMMENT '当前接口定义的全部内容版本摘要',
    `flag`        TINYINT(4)            DEFAULT 0 COMMENT '逻辑删除标志位, 0 位有效, 1 为逻辑删除',
    `ctime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    -- 服务契约id + method + path + source 需保证唯一
    KEY (`contract_id`, `path`, `method`, `source`)
) ENGINE = InnoDB;

/* 灰度资源 */
CREATE TABLE `gray_resource`
(
    `name`        VARCHAR(128)    NOT NULL COMMENT '灰度资源',
    `match_rule`  TEXT            NOT NULL COMMENT '配置规则',
    `create_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by`   VARCHAR(32)     DEFAULT "" COMMENT '创建人',
    `modify_time` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by`   VARCHAR(32)     DEFAULT "" COMMENT '最后更新人',
    `flag`        TINYINT(4)            DEFAULT 0 COMMENT '逻辑删除标志位, 0 位有效, 1 为逻辑删除',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB COMMENT = '灰度资源表';
