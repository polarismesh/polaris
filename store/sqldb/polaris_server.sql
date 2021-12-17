-- phpMyAdmin SQL Dump
-- version 4.6.4
-- https://www.phpmyadmin.net/
--
-- Host: 127.0.0.1
-- Generation Time: 2019-09-30 03:19:00
-- 服务器版本： 5.7.14
-- PHP Version: 5.6.25

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `polaris_server`
--
CREATE DATABASE IF NOT EXISTS `polaris_server` DEFAULT CHARACTER SET utf8 COLLATE utf8_bin;
USE `polaris_server`;

-- --------------------------------------------------------
--
-- 表的结构 `business`
--
CREATE TABLE `business` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '唯一ID',
    `name` varchar(64) COLLATE utf8_bin NOT NULL comment '业务名称',
    `token` varchar(64) COLLATE utf8_bin NOT NULL comment '该业务的token标识',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment '该业务的负责owner',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `instance`
--
CREATE TABLE `instance` (
    `id` varchar(128) COLLATE utf8_bin NOT NULL comment '唯一ID',
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `vpc_id` varchar(64) COLLATE utf8_bin DEFAULT NULL comment 'VPC ID',
    `host` varchar(128) COLLATE utf8_bin NOT NULL comment '实例的Host信息',
    `port` int(11) NOT NULL comment '实例的端口信息',
    `protocol` varchar(32) COLLATE utf8_bin DEFAULT NULL comment '对应端口的监听协议，比如tpc、udp、grpc、dubbo等等',
    `version` varchar(32) COLLATE utf8_bin DEFAULT NULL comment '实例的版本，可以用于版本路由',
    `health_status` tinyint(4) NOT NULL DEFAULT '1' comment '实例的健康状态，1为健康，0为不健康',
    `isolate` tinyint(4) NOT NULL DEFAULT '0' comment '实例隔离状态标志位，0为未隔离，1为隔离',
    `weight` smallint(6) NOT NULL DEFAULT '100' comment '实例的权重，主要用于loadbalance，默认为100',
    `enable_health_check` tinyint(4) NOT NULL DEFAULT '0' comment '是否对实例开启心跳上报检查逻辑，0为不开启，1为开启',
    `logic_set` varchar(128) COLLATE utf8_bin DEFAULT NULL comment '实例的逻辑分组信息',
    `cmdb_region` varchar(128) COLLATE utf8_bin DEFAULT NULL comment '实例的region信息，主要用于就近路由',
    `cmdb_zone` varchar(128) COLLATE utf8_bin DEFAULT NULL comment '实例的zone信息，主要用于就近路由',
    `cmdb_idc` varchar(128) COLLATE utf8_bin DEFAULT NULL comment '实例的IDC信息，主要用于就近路由',
    `priority` tinyint(4) NOT NULL DEFAULT '0' comment '实例的优先级，目前暂无用处',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment '实例的版本信息',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `service_id` (`service_id`),
    KEY `mtime` (`mtime`),
    KEY `host` (`host`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `health_check`
--
CREATE TABLE `health_check` (
    `id` varchar(128) COLLATE utf8_bin NOT NULL comment '实例ID',
    `type` tinyint(4) NOT NULL DEFAULT '0' comment '实例的健康检查类型',
    `ttl` int(11) NOT NULL comment '心跳的TTL时间',
    PRIMARY KEY (`id`),
    CONSTRAINT `health_check_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `instance_metadata`
--
CREATE TABLE `instance_metadata` (
    `id` varchar(128) COLLATE utf8_bin NOT NULL comment '实例ID',
    `mkey` varchar(128) COLLATE utf8_bin NOT NULL comment '实例标签的key',
    `mvalue` varchar(4096) COLLATE utf8_bin NOT NULL comment '实例标签的value',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`, `mkey`),
    KEY `mkey` (`mkey`),
    CONSTRAINT `instance_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `namespace`
--
CREATE TABLE `namespace` (
    `name` varchar(64) COLLATE utf8_bin NOT NULL comment '命名空间名称，唯一',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '命名空间的描述',
    `token` varchar(64) COLLATE utf8_bin NOT NULL comment '命名空间的token，用于写操作检查',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment '命名空间的负责owner',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

--
-- 转存表中的数据 `namespace`
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
-- 表的结构 `routing_config`
--
CREATE TABLE `routing_config` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '路由配置ID',
    `in_bounds` text COLLATE utf8_bin comment '服务被调路由规则',
    `out_bounds` text COLLATE utf8_bin comment '服务主调路由规则',
    `revision` varchar(40) COLLATE utf8_bin NOT NULL comment '路由规则版本',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment  '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `ratelimit_config`
--
CREATE TABLE `ratelimit_config` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '限流规则ID',
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `cluster_id` varchar(32) COLLATE utf8_bin NOT NULL comment '集群ID, 暂无使用',
    `labels` text COLLATE utf8_bin NOT NULL comment '针对特定的标签进行限流',
    `priority` smallint(6) NOT NULL DEFAULT '0' comment '限流规则优先级',
    `rule` text COLLATE utf8_bin NOT NULL comment '限流规则',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment '限流版本',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment  '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`),
    KEY `service_id` (`service_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `ratelimit_revision`
--
CREATE TABLE `ratelimit_revision` (
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `last_revision` varchar(40) COLLATE utf8_bin NOT NULL comment '对应服务的最新限流规则版本',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`service_id`),
    KEY `service_id` (`service_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `service`
--
CREATE TABLE `service` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `name` varchar(128) COLLATE utf8_bin NOT NULL comment '服务名称，命名空间下唯一',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL comment '服务所属的namespace',
    `ports` varchar(8192) COLLATE utf8_bin DEFAULT NULL comment '服务会对外暴露的所有端口信息列表（单个进程暴露多种协议）',
    `business` varchar(64) COLLATE utf8_bin DEFAULT NULL comment '服务的业务信息',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '服务的部门信息',
    `cmdb_mod1` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `cmdb_mod2` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `cmdb_mod3` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '服务的描述信息',
    `token` varchar(2048) COLLATE utf8_bin NOT NULL comment '服务的token，用于处理所有该服务涉及的写动作',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment '服务的版本信息',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment '服务所属的owner信息',
    `flag` tinyint(4) NOT NULL DEFAULT '0'  comment  '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `reference` varchar(32) COLLATE utf8_bin DEFAULT NULL comment '服务别名，表示该服务实际指向的服务名称是什么',
    `refer_filter` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '',
    `platform_id` varchar(32) COLLATE utf8_bin DEFAULT '' comment '服务所属的平台ID',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`, `namespace`),
    KEY `namespace` (`namespace`),
    KEY `mtime` (`mtime`),
    KEY `reference` (`reference`),
    KEY `platform_id` (`platform_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 转存表中的数据 `service`
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
-- 表的结构 `service_metadata`
--
CREATE TABLE `service_metadata` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `mkey` varchar(128) COLLATE utf8_bin NOT NULL comment '服务标签的key',
    `mvalue` varchar(4096) COLLATE utf8_bin NOT NULL comment '服务标签的value',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`, `mkey`),
    KEY `mkey` (`mkey`),
    CONSTRAINT `service_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `owner_service_map`，快速查询某个owner下的所有服务
--
CREATE TABLE `owner_service_map` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '',
    `owner` varchar(32) COLLATE utf8_bin NOT NULL comment '服务owner',
    `service` varchar(128) COLLATE utf8_bin NOT NULL comment '服务名称',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL,
    PRIMARY KEY (`id`),
    KEY `owner` (`owner`),
    KEY `name` (`service`, `namespace`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `circuitbreaker_rule`
--
CREATE TABLE `circuitbreaker_rule` (
    `id` varchar(97) COLLATE utf8_bin NOT NULL comment '熔断规则ID',
    `version` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT 'master' comment '熔断规则版本，默认为 mastr',
    `name` varchar(32) COLLATE utf8_bin NOT NULL comment '熔断规则名称',
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL comment '熔断规则所属命名空间',
    `business` varchar(64) COLLATE utf8_bin DEFAULT NULL comment '熔断规则的业务信息',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '熔断规则所属的部门信息',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '熔断规则的描述',
    `inbounds` text COLLATE utf8_bin NOT NULL comment '服务被调的熔断规则',
    `outbounds` text COLLATE utf8_bin NOT NULL comment '服务主调的熔断规则',
    `token` varchar(32) COLLATE utf8_bin NOT NULL comment '熔断规则的token，主要用于写操作检查',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment '熔断规则owner信息',
    `revision` varchar(32) COLLATE utf8_bin NOT NULL comment '熔断规则版本信息',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment  '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`, `version`),
    UNIQUE KEY `name` (`name`, `namespace`, `version`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `circuitbreaker_rule_relation`
--
CREATE TABLE `circuitbreaker_rule_relation` (
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `rule_id` varchar(97) COLLATE utf8_bin NOT NULL comment '熔断规则ID',
    `rule_version` varchar(32) COLLATE utf8_bin NOT NULL comment '熔断规则版本',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`service_id`),
    KEY `mtime` (`mtime`),
    KEY `rule_id` (`rule_id`),
    CONSTRAINT `circuitbreaker_rule_relation_ibfk_1` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `platform`
--
CREATE TABLE `platform` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL comment '平台ID',
    `name` varchar(128) COLLATE utf8_bin NOT NULL comment '平台名称',
    `domain` varchar(1024) COLLATE utf8_bin NOT NULL comment '平台域名',
    `qps` smallint(6) NOT NULL comment '针对某一平台设置的qps限制',
    `token` varchar(32) COLLATE utf8_bin NOT NULL comment '平台token',
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL comment '平台负责owner',
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '平台部门',
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL comment '平台描述',
    `flag` tinyint(4) NOT NULL DEFAULT '0' comment '逻辑删除标志位，0表示可见，1表示已被逻辑删除',
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `t_ip_config`
--
CREATE TABLE `t_ip_config` (
    `Fip` int(10) unsigned NOT NULL comment '机器IP',
    `FareaId` int(10) unsigned NOT NULL comment '区域编号',
    `FcityId` int(10) unsigned NOT NULL comment '城市编号',
    `FidcId` int(10) unsigned NOT NULL comment 'IDC编号',
    `Fflag` tinyint(4) DEFAULT '0',
    `Fstamp` datetime NOT NULL,
    `Fflow` int(10) unsigned NOT NULL,
    PRIMARY KEY (`Fip`),
    KEY `idx_Fflow` (`Fflow`)
) ENGINE = InnoDB DEFAULT CHARSET = latin1;

-- --------------------------------------------------------
--
-- 表的结构 `t_policy`
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
-- 表的结构 `t_route`
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
-- 表的结构 `t_section`
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
-- 表的结构 `start_lock`
--
CREATE TABLE `start_lock` (
    `lock_id` int(11) NOT NULL COMMENT '锁序号',
    `lock_key` varchar(32) COLLATE utf8_bin NOT NULL COMMENT '锁的名字',
    `server` varchar(32) COLLATE utf8_bin NOT NULL COMMENT '持有启动锁的Server',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`lock_id`, `lock_key`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

--
-- 转存表中的数据 `start_lock`
--
INSERT INTO
    `start_lock` (`lock_id`, `lock_key`, `server`, `mtime`)
VALUES
    (1, 'sz', 'aaa', '2019-12-05 08:35:49');

-- --------------------------------------------------------
--
-- 表的结构 `cl5_module`
--
CREATE TABLE `cl5_module` (
    `module_id` int(11) NOT NULL COMMENT '模块ID',
    `interface_id` int(11) NOT NULL COMMENT '接口ID',
    `range_num` int(11) NOT NULL,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`module_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin COMMENT = '用以生成sid';

--
-- 转存表中的数据 `cl5_module`
--
insert into
    cl5_module(module_id, interface_id, range_num)
values
(3000001, 1, 0);

-- --------------------------------------------------------
--
-- 表的结构 `mesh`
--
CREATE TABLE `mesh` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*网格ID*/
    `name` varchar(128) COLLATE utf8_bin NOT NULL,
    /*网格名*/
    `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
    /*网格所属部门*/
    `business` varchar(128) COLLATE utf8_bin NOT NULL,
    /*网格所属业务*/
    `managed` tinyint(4) NOT NULL,
    /*是否托管*/
    `istio_version` varchar(64) COLLATE utf8_bin,
    /*istio版本*/
    `data_cluster` varchar(1024) COLLATE utf8_bin,
    /*数据面集群*/
    `revision` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则版本号*/
    `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
    /*规则描述*/
    `token` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则鉴权token*/
    `owner` varchar(1024) COLLATE utf8_bin NOT NULL,
    /*规则的拥有者*/
    `flag` tinyint(4) NOT NULL DEFAULT '0',
    /*规则是否有效，0为有效，1为无效，己被删除了*/
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `mesh_service`
--
CREATE TABLE `mesh_service` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*网格规则ID*/
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*网格名*/
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*服务ID*/
    `namespace` varchar(64) COLLATE utf8_bin NOT NULL,
    /*服务命名空间*/
    `service` varchar(128) COLLATE utf8_bin NOT NULL,
    /*服务名*/
    `mesh_namespace` varchar(64) COLLATE utf8_bin NOT NULL,
    /*映射到网格的命名空间*/
    `mesh_service` varchar(128) COLLATE utf8_bin NOT NULL,
    /*映射到网格的服务名*/
    `location` varchar(16) COLLATE utf8_bin NOT NULL,
    /*服务处于网格哪个位置*/
    `export_to` varchar(1024) COLLATE utf8_bin NOT NULL,
    /*服务可以被哪些命名空间所见*/
    `revision` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则版本号*/
    `flag` tinyint(4) NOT NULL DEFAULT '0',
    /*规则是否有效，0为有效，1为无效，己被删除了*/
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
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
-- 表的结构 `mesh_service_revision`
--
CREATE TABLE `mesh_service_revision` (
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*网格名*/
    `revision` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则版本号*/
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`mesh_id`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- 表的结构 `mesh_resource`
--
CREATE TABLE `mesh_resource` (
    `id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*网格规则ID*/
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*网格名*/
    `name` varchar(64) COLLATE utf8_bin NOT NULL,
    /*规则名*/
    `mesh_namespace` varchar(64) COLLATE utf8_bin NOT NULL,
    /*规则所处的网格命名空间*/
    `type_url` varchar(96) COLLATE utf8_bin NOT NULL,
    /*规则类型，如virtualService*/
    `revision` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则版本号*/
    `body` text,
    /*规则内容，json格式字符串*/
    `flag` tinyint(4) NOT NULL DEFAULT '0',
    /*规则是否有效，0为有效，1为无效，己被删除了*/
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name`(`mesh_id`, `name`, `mesh_namespace`, `type_url`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

--
-- 表的结构 `mesh_revision`
--
CREATE TABLE `mesh_resource_revision` (
    `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则所属网格ID*/
    `type_url` varchar(96) COLLATE utf8_bin NOT NULL,
    /*规则类型，如virtualService*/
    `revision` varchar(32) COLLATE utf8_bin NOT NULL,
    /*规则集合的版本号，同一个网格下面所有规则集合的总体版本号*/
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`mesh_id`, `type_url`),
    KEY `mtime` (`mtime`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
--
-- flux规则配置表的结构 `ratelimit_flux_rule_config`
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
    `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
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
-- flux规则版本关联表的结构 `ratelimit_flux_rule_revision`
--
CREATE TABLE `ratelimit_flux_rule_revision` (
    `service_id` varchar(32) COLLATE utf8_bin NOT NULL comment '服务ID',
    `last_revision` varchar(40) COLLATE utf8_bin NOT NULL comment 'flux规则的最新版本',
    `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`service_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8 COLLATE = utf8_bin;

-- --------------------------------------------------------
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
