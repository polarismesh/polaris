SET
    SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";

SET
    time_zone = "+00:00";

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */
;

/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */
;

/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */
;

/*!40101 SET NAMES utf8mb4 */
;

--
-- Database: `polaris_server`
--
CREATE DATABASE IF NOT EXISTS `polaris_server` DEFAULT CHARACTER SET utf8 COLLATE utf8_bin;

USE `polaris_server`;

-- --------------------------------------------------------
--
-- Table structure `business`
--
CREATE TABLE `business` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Unique ID',
    `name` varchar(64) COLLATE utf8_bin NOT NULL comment 'business name',
    `token` varchar(64) COLLATE utf8_bin NOT NULL comment 'Token ID of the business',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment 'The business is responsible for Owner',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `instance`
--
CREATE TABLE `instance` (
    `id` varchar(128) COLLATE utf8_bin NOT NULL comment 'Unique ID',
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `vpc_id` varchar(64) COLLATE utf8_bin DEFAULT NULL comment 'VPC ID',
    `host` varchar(128) COLLATE utf8_bin NOT NULL comment 'instance Host Information',
    `port` int(11) NOT NULL comment 'instance port information',
    `protocol` varchar(32) COLLATE utf8_bin DEFAULT NULL comment 'Listening protocols for corresponding ports, such as TPC, UDP, GRPC, DUBBO, etc.',
    `version` varchar(32) COLLATE utf8_bin DEFAULT NULL comment 'The version of the instance can be used for version routing',
    `health_status` tinyint(4) NOT NULL DEFAULT '1' comment 'The health status of the instance, 1 is health, 0 is unhealthy',
    `isolate` tinyint(4) NOT NULL DEFAULT '0' comment 'Example isolation status flag, 0 is not isolated, 1 is isolated',
    `weight` smallint(6) NOT NULL DEFAULT '100' comment 'The weight of the instance is mainly used for LoadBalance, default is 100',
    `enable_health_check` tinyint(4) NOT NULL DEFAULT '0' comment 'Whether to open a heartbeat on an instance, check the logic, 0 is not open, 1 is open',
    `logic_set` varchar(128) COLLATE utf8_bin DEFAULT NULL comment 'Example logic packet information',
    `cmdb_region` varchar(128) COLLATE utf8_bin DEFAULT NULL comment 'The region information of the instance is mainly used to close the route',
    `cmdb_zone` varchar(128) COLLATE utf8_bin DEFAULT NULL comment 'The ZONE information of the instance is mainly used to close the route.',
    `cmdb_idc` varchar(128) COLLATE utf8_bin DEFAULT NULL comment 'The IDC information of the instance is mainly used to close the route',
    `priority` tinyint(4) NOT NULL DEFAULT '0' comment 'Example priority, currently useless',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment 'Instance version information',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `service_id` (`service_id`),
    KEY `mtime` (`mtime`),
    KEY `host` (`host`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `health_check`
--
CREATE TABLE `health_check` (
    `id` varchar(128) COLLATE utf8_bin NOT NULL comment 'Instance ID',
    `type` tinyint(4) NOT NULL DEFAULT '0' comment 'Instance health check type',
    `ttl` int(11) NOT NULL comment 'TTL time jumping',
    PRIMARY KEY (`id`),
    CONSTRAINT `health_check_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `instance_metadata`
--
CREATE TABLE `instance_metadata` (
    `id` varchar(128) COLLATE utf8_bin NOT NULL comment 'Instance ID',
    `mkey` varchar(128) COLLATE utf8_bin NOT NULL comment 'instance label of Key',
    `mvalue` varchar(4096) COLLATE utf8_bin NOT NULL comment 'instance label Value',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`, `mkey`),
    KEY `mkey` (`mkey`),
    CONSTRAINT `instance_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `namespace`
--
CREATE TABLE `namespace` (
    `name` varchar(64) COLLATE utf8_bin NOT NULL comment 'Namespace name, unique',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Description of namespace',
    `token` varchar(64) COLLATE utf8_bin NOT NULL comment 'TOKEN named space for write operation check',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment 'Responsible for named space Owner',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

--
-- Data in the conveyor `namespace`
--
INSERT INTO
    `namespace` (
        `name`,
        `comment`,
        `token`,
        `owner`,
        `flag`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'Polaris',
        'Polaris-server',
        '2d1bfe5d12e04d54b8ee69e62494c7fd',
        'polaris',
        0,
        '2019-09-06 07:55:07',
        '2019-09-06 07:55:07'
    ),
    (
        'default',
        'Default Environment',
        'e2e473081d3d4306b52264e49f7ce227',
        'polaris',
        0,
        '2021-07-27 19:37:37',
        '2021-07-27 19:37:37'
    );

-- --------------------------------------------------------
--
-- Table structure `routing_config`
--
CREATE TABLE `routing_config` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Routing configuration ID',
    `in_bounds` text COLLATE utf8_bin comment 'Service is routing rules',
    `out_bounds` text COLLATE utf8_bin comment 'Service main routing rules',
    `revision` varchar(40) COLLATE utf8_bin NOT NULL comment 'Routing rule version',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `ratelimit_config`
--
CREATE TABLE `ratelimit_config` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment 'ratelimit rule ID',
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `cluster_id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Cluster ID, no use',
    `labels` text COLLATE utf8_bin NOT NULL comment 'Conductive flow for a specific label',
    `priority` smallint(6) NOT NULL DEFAULT '0' comment 'ratelimit rule priority',
    `rule` text COLLATE utf8_bin NOT NULL comment 'Current limiting rules',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment 'Limiting version',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`),
    KEY `service_id` (`service_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `ratelimit_revision`
--
CREATE TABLE `ratelimit_revision` (
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `last_revision` varchar(40) COLLATE utf8_bin NOT NULL comment 'The latest limited limiting rule version of the corresponding service',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`service_id`),
    KEY `service_id` (`service_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `service`
--
CREATE TABLE `service` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `name` varchar(128) COLLATE utf8_bin NOT NULL comment 'Service name, only under the namespace',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL comment 'Namespace belongs to the service',
    `ports` varchar(8192) COLLATE utf8_bin DEFAULT NULL comment 'Service will have a list of all port information of the external exposure (single process exposing multiple protocols)',
    `business` varchar(64) COLLATE utf8_bin DEFAULT NULL comment 'Service business information',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Service department information',
    `cmdb_mod1` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `cmdb_mod2` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `cmdb_mod3` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Description information',
    `token` varchar(2048) COLLATE utf8_bin NOT NULL comment 'Service token, used to handle all the services involved in the service',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service version information',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment 'Owner information belonging to the service',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `reference` varchar(32) COLLATE utf8_bin DEFAULT NULL comment 'Service alias, what is the actual service name that the service is actually pointed out?',
    `refer_filter` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `platform_id` varchar(32) COLLATE utf8_bin DEFAULT '' comment 'The platform ID to which the service belongs',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`, `namespace`),
    KEY `namespace` (`namespace`),
    KEY `mtime` (`mtime`),
    KEY `reference` (`reference`),
    KEY `platform_id` (`platform_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Data in the conveyor `service`
--
INSERT INTO
    `service` (
        `id`,
        `name`,
        `namespace`,
        `comment`,
        `business`,
        `token`,
        `revision`,
        `owner`,
        `flag`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        'polaris.checker',
        'Polaris',
        'polaris checker service',
        'polaris',
        '7d19c46de327408d8709ee7392b7700b',
        '301b1e9f0bbd47a6b697e26e99dfe012',
        'polaris',
        0,
        '2021-09-06 07:55:07',
        '2021-09-06 07:55:09'
    ),
    (
        'bbfdda174ea64e11ac862adf14593c03',
        'polaris.monitor',
        'Polaris',
        'polaris monitor service',
        'polaris',
        '50b4e7d8affa4634b52523d398d1a369',
        '3649b17283d94d7baee5fb5d8160a225',
        'polaris',
        0,
        '2021-09-06 07:55:07',
        '2021-09-06 07:55:11'
    );

-- --------------------------------------------------------
--
-- Table structure `service_metadata`
--
CREATE TABLE `service_metadata` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `mkey` varchar(128) COLLATE utf8_bin NOT NULL comment 'Service label key',
    `mvalue` varchar(4096) COLLATE utf8_bin NOT NULL comment 'Service label Value',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`, `mkey`),
    KEY `mkey` (`mkey`),
    CONSTRAINT `service_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `owner_service_map`Quickly query all services under an Owner
--
CREATE TABLE `owner_service_map` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '',
    `owner` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service Owner',
    `service` varchar(128) COLLATE utf8_bin NOT NULL comment 'service name',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL comment 'namespace name',
    PRIMARY KEY (`id`),
    KEY `owner` (`owner`),
    KEY `name` (`service`, `namespace`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `circuitbreaker_rule`
--
CREATE TABLE `circuitbreaker_rule` (
    `id` varchar(97) COLLATE utf8_bin NOT NULL comment 'Melting rule ID',
    `version` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT 'master' comment 'Melting rule version, default is MASTR',
    `name` varchar(32) COLLATE utf8_bin NOT NULL comment 'Melting rule name',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL comment 'Melting rule belongs to name space',
    `business` varchar(64) COLLATE utf8_bin DEFAULT NULL comment 'Business information of fuse regular',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Department information to which the fuse regular belongs',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Description of the fuse rule',
    `inbounds` text COLLATE utf8_bin NOT NULL comment 'Service-tuned fuse rule',
    `outbounds` text COLLATE utf8_bin NOT NULL comment 'Service Motoring Fuse Rule',
    `token` varchar(32) COLLATE utf8_bin NOT NULL comment 'Token, which is fucking, mainly for writing operation check',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment 'Melting rule Owner information',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment 'Melt rule version information',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`, `version`),
    UNIQUE KEY `name` (`name`, `namespace`, `version`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `circuitbreaker_rule_relation`
--
CREATE TABLE `circuitbreaker_rule_relation` (
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `rule_id` varchar(97) COLLATE utf8_bin NOT NULL comment 'Melting rule ID',
    `rule_version` varchar(32) COLLATE utf8_bin NOT NULL comment 'Melting rule version',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`service_id`),
    KEY `mtime` (`mtime`),
    KEY `rule_id` (`rule_id`),
    CONSTRAINT `circuitbreaker_rule_relation_ibfk_1` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `platform`
--
CREATE TABLE `platform` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Platform ID',
    `name` varchar(128) COLLATE utf8_bin NOT NULL comment 'Platform name',
    `domain` varchar(1024) COLLATE utf8_bin NOT NULL comment 'Platform domain name',
    `qps` smallint(6) NOT NULL comment 'QPS restrictions set for a platform',
    `token` varchar(32) COLLATE utf8_bin NOT NULL comment 'Platform token',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment 'Platform is responsible for Owner',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Platform department',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment 'Platform description',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment 'Logic delete flag, 0 means visible, 1 means that it has been logically deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `t_ip_config`
--
CREATE TABLE `t_ip_config` (
    `Fip` int(10) unsigned NOT NULL comment 'Machine IP',
    `FareaId` int(10) unsigned NOT NULL comment 'Area number',
    `FcityId` int(10) unsigned NOT NULL comment 'City number',
    `FidcId` int(10) unsigned NOT NULL comment 'IDC number',
    `Fflag` tinyint(4) DEFAULT '0',
    `Fstamp` datetime NOT NULL,
    `Fflow` int(10) unsigned NOT NULL,
    PRIMARY KEY (`Fip`),
    KEY `idx_Fflow` (`Fflow`)
) ENGINE = InnoDB DEFAULT CHARSET = latin1;

-- --------------------------------------------------------
--
-- Table structure `t_policy`
--
CREATE TABLE `t_policy` (
    `FmodId` int(10) unsigned NOT NULL,
    `Fdiv` int(10) unsigned NOT NULL,
    `Fmod` int(10) unsigned NOT NULL,
    `Fflag` tinyint(4) DEFAULT '0',
    `Fstamp` datetime NOT NULL,
    `Fflow` int(10) unsigned NOT NULL,
    PRIMARY KEY (`FmodId`)
) ENGINE = InnoDB DEFAULT CHARSET = latin1;

-- --------------------------------------------------------
--
-- Table structure `t_route`
--
CREATE TABLE `t_route` (
    `Fip` int(10) unsigned NOT NULL,
    `FmodId` int(10) unsigned NOT NULL,
    `FcmdId` int(10) unsigned NOT NULL,
    `FsetId` varchar(32) NOT NULL,
    `Fflag` tinyint(4) DEFAULT '0',
    `Fstamp` datetime NOT NULL,
    `Fflow` int(10) unsigned NOT NULL,
    PRIMARY KEY (`Fip`, `FmodId`, `FcmdId`),
    KEY `Fflow` (`Fflow`),
    KEY `idx1` (`FmodId`, `FcmdId`, `FsetId`)
) ENGINE = InnoDB DEFAULT CHARSET = latin1;

-- --------------------------------------------------------
--
-- Table structure `t_section`
--
CREATE TABLE `t_section` (
    `FmodId` int(10) unsigned NOT NULL,
    `Ffrom` int(10) unsigned NOT NULL,
    `Fto` int(10) unsigned NOT NULL,
    `Fxid` int(10) unsigned NOT NULL,
    `Fflag` tinyint(4) DEFAULT '0',
    `Fstamp` datetime NOT NULL,
    `Fflow` int(10) unsigned NOT NULL,
    PRIMARY KEY (`FmodId`, `Ffrom`, `Fto`)
) ENGINE = InnoDB DEFAULT CHARSET = latin1;

-- --------------------------------------------------------
--
-- Table structure `start_lock`
--
CREATE TABLE `start_lock` (
    `lock_id` int(11) NOT NULL COMMENT '锁序号',
    `lock_key` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Lock name',
    `server` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'SERVER holding launch lock',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    PRIMARY KEY (`lock_id`, `lock_key`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

--
-- Data in the conveyor `start_lock`
--
INSERT INTO
    `start_lock` (`lock_id`, `lock_key`, `server`, `mtime`)
VALUES
    (1, 'sz', 'aaa', '2019-12-05 08:35:49');

-- --------------------------------------------------------
--
-- Table structure `cl5_module`
--
CREATE TABLE `cl5_module` (
    `module_id` int(11) NOT NULL COMMENT 'Module ID',
    `interface_id` int(11) NOT NULL COMMENT 'Interface ID',
    `range_num` int(11) NOT NULL,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`module_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin COMMENT = 'To generate SID';

--
-- Data in the conveyor `cl5_module`
--
insert into
    cl5_module(module_id, interface_id, range_num)
values
    (3000001, 1, 0);

-- --------------------------------------------------------
--
-- Table structure `mesh`
--
CREATE TABLE `mesh` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'mesh ID',
    `name` varchar(128) COLLATE utf8_bin NOT NULL COMMENT 'mesh name',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL COMMENT 'mesh department',
    `business` varchar(128) COLLATE utf8_bin NOT NULL COMMENT 'mesh service',
    `managed` tinyint(4) NOT NULL COMMENT 'Whether to managed',
    `istio_version` varchar(64) COLLATE utf8_bin COMMENT 'ISTIO version',
    `data_cluster` varchar(1024) COLLATE utf8_bin COMMENT 'Data surface cluster',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Rule version number',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL COMMENT 'Rule description',
    `token` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Rule Authentication Token',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL COMMENT 'Rule owner',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `mesh_service`
--
CREATE TABLE `mesh_service` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'mesh rule ID',
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'mesh name',
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Service ID',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL COMMENT 'Service namespace',
    `service` varchar(128) COLLATE utf8_bin NOT NULL COMMENT 'Service Name',
    `mesh_namespace` varchar(64) COLLATE utf8_bin NOT NULL COMMENT 'Map to the mesh namespace',
    `mesh_service` varchar(128) COLLATE utf8_bin NOT NULL COMMENT 'Mapping to the mesh service name',
    `location` varchar(16) COLLATE utf8_bin NOT NULL COMMENT 'Which location is in the mesh',
    `export_to` varchar(1024) COLLATE utf8_bin NOT NULL COMMENT 'What is the service you can be seen by the namespace',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Rule version number',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY `relation` (`mesh_id`, `mesh_namespace`, `mesh_service`),
    KEY `namespace`(`namespace`),
    KEY `service`(`service`),
    KEY `location`(`location`),
    KEY `export_to`(`export_to`),
    KEY `mtime` (`mtime`),
    KEY `flag`(`flag`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `mesh_service_revision`
--
CREATE TABLE `mesh_service_revision` (
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'mesh name',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Rule version number',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`mesh_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- Table structure `mesh_resource`
--
CREATE TABLE `mesh_resource` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'mesh rule ID',
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'mesh name',
    `name` varchar(64) COLLATE utf8_bin NOT NULL COMMENT 'Rule name',
    `mesh_namespace` varchar(64) COLLATE utf8_bin NOT NULL COMMENT 'mesh namespace where the rules are located',
    `type_url` varchar(96) COLLATE utf8_bin NOT NULL COMMENT 'Rule type, such as VirtualService',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Rule version number',
    `body` text COMMENT 'Rule content, JSON format string',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY `name`(`mesh_id`, `name`, `mesh_namespace`, `type_url`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

--
-- Table structure `mesh_revision`
--
CREATE TABLE `mesh_resource_revision` (
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'Rules, mesh ID',
    `type_url` varchar(96) COLLATE utf8_bin NOT NULL COMMENT 'Rule type, such as VirtualService',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL COMMENT 'The version number of the rules collection, the overall version number of all rule collections below the same mesh',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`mesh_id`, `type_url`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- FLUX Rule Configuring Table structure `ratelimit_flux_rule_config`
--
CREATE TABLE `ratelimit_flux_rule_config` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL,
    `revision` varchar(32) COLLATE utf8_bin NOT NULL,
    `callee_service_id` varchar(32) COLLATE utf8_bin NOT NULL,
    `callee_service_env` varchar(64) COLLATE utf8_bin NOT NULL,
    `callee_service_name` varchar(250) COLLATE utf8_bin NOT NULL DEFAULT '',
    `caller_service_business` varchar(250) COLLATE utf8_bin NOT NULL DEFAULT '',
    `name` varchar(100) COLLATE utf8_bin NOT NULL DEFAULT '',
    `description` varchar(500) COLLATE utf8_bin NOT NULL DEFAULT '',
    `type` tinyint(4) NOT NULL DEFAULT '0',
    `set_key` varchar(250) COLLATE utf8_bin NOT NULL DEFAULT '',
    `set_alert_qps` varchar(10) NOT NULL DEFAULT '',
    `set_warning_qps` varchar(10) NOT NULL DEFAULT '',
    `set_remark` varchar(500) COLLATE utf8_bin NOT NULL DEFAULT '',
    `default_key` varchar(250) COLLATE utf8_bin NOT NULL DEFAULT '',
    `default_alert_qps` varchar(10) NOT NULL DEFAULT '',
    `default_warning_qps` varchar(10) NOT NULL DEFAULT '',
    `default_remark` varchar(500) COLLATE utf8_bin NOT NULL DEFAULT '',
    `creator` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT '',
    `updater` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT '',
    `status` tinyint(4) NOT NULL DEFAULT '0',
    `flag` tinyint(4) NOT NULL DEFAULT '0',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    `flux_server_id` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT '',
    `monitor_server_id` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT '',
    PRIMARY KEY (`id`),
    UNIQUE KEY `unique_service` (
        `callee_service_id`,
        `caller_service_business`,
        `set_key`
    ),
    KEY `mtime` (`mtime`),
    KEY `name` (`name`),
    KEY `creator` (`creator`),
    KEY `callee_service` (`callee_service_env`, `callee_service_name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- FLUX rule version is associated with TABLE structure `ratelimit_flux_rule_revision`
--
CREATE TABLE `ratelimit_flux_rule_revision` (
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment 'Service ID',
    `last_revision` varchar(40) COLLATE utf8_bin NOT NULL comment 'Latest version of the FLUX rule',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`service_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

CREATE TABLE `user` (
    `id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '用户ID',
    `name` VARCHAR(100) COLLATE utf8_bin NOT NULL comment '用户名称',
    `password` VARCHAR(100) COLLATE utf8_bin NOT NULL comment '用户密码',
    `owner` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '主账户ID',
    `source` VARCHAR(32) COLLATE utf8_bin NOT NULL comment '账户来源',
    `token` VARCHAR(255) COLLATE utf8_bin NOT NULL comment '账户所拥有的 token 信息，可用于SDK访问鉴权',
    `token_enable` tinyint(4) NOT NULL DEFAULT 1,
    `user_type` int NOT NULL DEFAULT 20 comment 'Account type, 0 is the admin super account, 20 is the primary account, 50 for the child account',
    `comment` VARCHAR(255) COLLATE utf8_bin NOT NULL comment '描述',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

CREATE TABLE `user_group` (
    `id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '用户组ID',
    `name` VARCHAR(100) COLLATE utf8_bin NOT NULL comment '用户组名称',
    `owner` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '用户组所属的主账户ID',
    `token` VARCHAR(255) COLLATE utf8_bin NOT NULL comment '该用户组的 token 信息',
    `comment` VARCHAR(255) COLLATE utf8_bin NOT NULL comment '描述信息',
    `token_enable` tinyint(4) NOT NULL DEFAULT 1,
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

CREATE TABLE `user_group_relation` (
    `user_id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '用户ID',
    `group_id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '用户组ID',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`user_id`, `group_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

CREATE TABLE `auth_strategy` (
    `id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '策略ID',
    `name` VARCHAR(100) COLLATE utf8_bin NOT NULL comment '策略名称',
    `action` VARCHAR(32) COLLATE utf8_bin NOT NULL comment '该策略的读写权限, only_read = 0, read_write = 1',
    `owner` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '该策略所属的账号ID',
    `comment` VARCHAR(255) COLLATE utf8_bin NOT NULL comment '描述',
    `default` tinyint(4) NOT NULL DEFAULT '0',
    `revision` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '鉴权规则版本',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`id`),
    UNIQUE KEY (`name`, `owner`),
    KEY `owner` (`owner`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

CREATE TABLE `auth_principal` (
    `strategy_id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '策略ID',
    `principal_id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment 'principal的ID',
    `principal_role` int NOT NULL comment 'principal的类型, 1 为 User、2 为 Group',
    PRIMARY KEY (`strategy_id`, `principal_id`, `principal_role`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

CREATE TABLE `auth_strategy_resource` (
    `strategy_id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '策略ID',
    `res_type` int COLLATE utf8_bin NOT NULL comment '资源类型, Namespaces = 0, Service = 1, ConfigGroups = 2',
    `res_id` VARCHAR(128) COLLATE utf8_bin NOT NULL comment '资源ID',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Whether the rules are valid, 0 is valid, 1 is invalid, it is deleted',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'Create time',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'Last updated time',
    PRIMARY KEY (`strategy_id`, `res_type`, `res_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- 创建默认的超级账户，密码为 polarismesh@2021
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
        `owner`
    )
VALUES
    (
        '65e4789a6d5b49669adf1e9e8387549c',
        'polarisadmin',
        '$2a$10$5XMjs.oqo4PnpbTGy9dQqewL4eb4yoA7b/6ZKL33IPhFyIxzj4lRy',
        'Polaris',
        'nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=',
        1,
        0,
        'default polaris admin account',
        '65e4789a6d5b49669adf1e9e8387549c'
    );

-- 插入polaris-admin的权限策略
INSERT INTO
    `auth_strategy`(
        `id`,
        `name`,
        `action`,
        `owner`,
        `comment`,
        `default`,
        `revision`,
        `flag`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        '__default__user_65e4789a6d5b49669adf1e9e8387549c',
        'READ_WRITE',
        '65e4789a6d5b49669adf1e9e8387549c',
        'default admin',
        1,
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        sysdate(),
        sysdate()
    );

-- 插入 polaris-admin 可以访问的资源规则
INSERT INTO
    `auth_strategy_resource`(
        `strategy_id`,
        `res_type`,
        `res_id`,
        `flag`,
        `ctime`,
        `mtime`
    )
VALUES
    (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        0,
        '*',
        0,
        sysdate(),
        sysdate()
    );

-- 插入权限策略和polaris-admin账户的关联关系
INSERT INTO
    auth_principal(`strategy_id`, `principal_id`, `principal_role`) VALUE (
        'fbca9bfa04ae4ead86e1ecf5811e32a9',
        '65e4789a6d5b49669adf1e9e8387549c',
        1
    );