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
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `name` varchar(64) COLLATE utf8_bin NOT NULL,
  `token` varchar(64) COLLATE utf8_bin NOT NULL,
  `owner` varchar(1024) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `instance`
--

CREATE TABLE `instance` (
  `id` varchar(40) COLLATE utf8_bin NOT NULL,
  `service_id` varchar(32) COLLATE utf8_bin NOT NULL,
  `vpc_id` varchar(64) COLLATE utf8_bin DEFAULT NULL,
  `host` varchar(128) COLLATE utf8_bin NOT NULL,
  `port` int(11) NOT NULL,
  `protocol` varchar(32) COLLATE utf8_bin DEFAULT NULL,
  `version` varchar(32) COLLATE utf8_bin DEFAULT NULL,
  `health_status` tinyint(4) NOT NULL DEFAULT '1',
  `isolate` tinyint(4) NOT NULL DEFAULT '0',
  `weight` smallint(6) NOT NULL DEFAULT '100',
  `enable_health_check` tinyint(4) NOT NULL DEFAULT '0',
  `logic_set` varchar(128) COLLATE utf8_bin DEFAULT NULL,
  `cmdb_region` varchar(128) COLLATE utf8_bin DEFAULT NULL,
  `cmdb_zone` varchar(128) COLLATE utf8_bin DEFAULT NULL,
  `cmdb_idc` varchar(128) COLLATE utf8_bin DEFAULT NULL,
  `priority` tinyint(4) NOT NULL DEFAULT '0',
  `revision` varchar(32) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `service_id` (`service_id`),
  KEY `mtime` (`mtime`),
  KEY `host` (`host`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `health_check`
--

CREATE TABLE `health_check` (
  `id` varchar(40) COLLATE utf8_bin NOT NULL,
  `type` tinyint(4) NOT NULL DEFAULT '0',
  `ttl` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `health_check_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `instance_metadata`
--

CREATE TABLE `instance_metadata` (
  `id` varchar(40) COLLATE utf8_bin NOT NULL,
  `mkey` varchar(128) COLLATE utf8_bin NOT NULL,
  `mvalue` varchar(4096) COLLATE utf8_bin NOT NULL,
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`mkey`),
  KEY `mkey` (`mkey`),
  CONSTRAINT `instance_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `instance` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `namespace`
--

CREATE TABLE `namespace` (
  `name` varchar(64) COLLATE utf8_bin NOT NULL,
  `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `token` varchar(64) COLLATE utf8_bin NOT NULL,
  `owner` varchar(1024) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- 转存表中的数据 `namespace`
--

INSERT INTO `namespace` (`name`, `comment`, `token`, `owner`, `flag`, `ctime`, `mtime`) VALUES
('Polaris', 'Polaris-server', '2d1bfe5d12e04d54b8ee69e62494c7fd', 'polaris', 0, '2019-09-06 07:55:07', '2019-09-06 07:55:07'),
('default', 'Default Environment', 'e2e473081d3d4306b52264e49f7ce227', 'polaris', 0, '2021-07-27 19:37:37', '2021-07-27 19:37:37');

-- --------------------------------------------------------

--
-- 表的结构 `routing_config`
--

CREATE TABLE `routing_config` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `in_bounds` text COLLATE utf8_bin,
  `out_bounds` text COLLATE utf8_bin,
  `revision` varchar(40) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `ratelimit_config`
--

CREATE TABLE `ratelimit_config` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `service_id` varchar(32) COLLATE utf8_bin NOT NULL,
  `cluster_id` varchar(32) COLLATE utf8_bin NOT NULL,
  `labels` text COLLATE utf8_bin NOT NULL,
  `priority` smallint(6) NOT NULL DEFAULT '0',
  `rule` text COLLATE utf8_bin NOT NULL,
  `revision` varchar(32) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `mtime` (`mtime`),
  KEY `service_id` (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `ratelimit_revision`
--

CREATE TABLE `ratelimit_revision` (
  `service_id` varchar(32) COLLATE utf8_bin NOT NULL,
  `last_revision` varchar(40) COLLATE utf8_bin NOT NULL,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`service_id`),
  KEY `service_id` (`service_id`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `service`
--

CREATE TABLE `service` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `name` varchar(128) COLLATE utf8_bin NOT NULL,
  `namespace` varchar(64) COLLATE utf8_bin NOT NULL,
  `ports` varchar(8192) COLLATE utf8_bin DEFAULT NULL,
  `business` varchar(64) COLLATE utf8_bin DEFAULT NULL,
  `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `cmdb_mod1` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `cmdb_mod2` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `cmdb_mod3` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `token` varchar(2048) COLLATE utf8_bin NOT NULL,
  `revision` varchar(32) COLLATE utf8_bin NOT NULL,
  `owner` varchar(1024) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `reference` varchar(32) COLLATE utf8_bin DEFAULT NULL,
  `refer_filter` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `platform_id` varchar(32) COLLATE utf8_bin DEFAULT '',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`,`namespace`),
  KEY `namespace` (`namespace`),
  KEY `mtime` (`mtime`),
  KEY `reference` (`reference`),
  KEY `platform_id` (`platform_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `service_metadata`
--

CREATE TABLE `service_metadata` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `mkey` varchar(128) COLLATE utf8_bin NOT NULL,
  `mvalue` varchar(4096) COLLATE utf8_bin NOT NULL,
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`mkey`),
  KEY `mkey` (`mkey`),
  CONSTRAINT `service_metadata_ibfk_1` FOREIGN KEY (`id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `owner_service_map`
--

CREATE TABLE `owner_service_map` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `owner` varchar(32) COLLATE utf8_bin NOT NULL,
  `service` varchar(128) COLLATE utf8_bin NOT NULL,
  `namespace` varchar(64) COLLATE utf8_bin NOT NULL,
  PRIMARY KEY (`id`),
  KEY `owner` (`owner`),
  KEY `name` (`service`,`namespace`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `circuitbreaker_rule`
--

CREATE TABLE `circuitbreaker_rule` (
  `id` varchar(97) COLLATE utf8_bin NOT NULL,
  `version` varchar(32) COLLATE utf8_bin NOT NULL DEFAULT 'master',
  `name` varchar(32) COLLATE utf8_bin NOT NULL,
  `namespace` varchar(64) COLLATE utf8_bin NOT NULL,
  `business` varchar(64) COLLATE utf8_bin DEFAULT NULL,
  `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `inbounds` text COLLATE utf8_bin NOT NULL,
  `outbounds` text COLLATE utf8_bin NOT NULL,
  `token` varchar(32) COLLATE utf8_bin NOT NULL,
  `owner` varchar(1024) COLLATE utf8_bin NOT NULL,
  `revision` varchar(32) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`,`version`),
  UNIQUE KEY `name` (`name`,`namespace`,`version`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `circuitbreaker_rule_relation`
--

CREATE TABLE `circuitbreaker_rule_relation` (
  `service_id` varchar(32) COLLATE utf8_bin NOT NULL,
  `rule_id` varchar(97) COLLATE utf8_bin NOT NULL,
  `rule_version` varchar(32) COLLATE utf8_bin NOT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`service_id`),
  KEY `mtime` (`mtime`),
  KEY `rule_id` (`rule_id`),
  CONSTRAINT `circuitbreaker_rule_relation_ibfk_1` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `platform`
--

CREATE TABLE `platform` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL,
  `name` varchar(128) COLLATE utf8_bin NOT NULL,
  `domain` varchar(1024) COLLATE utf8_bin NOT NULL,
  `qps` smallint(6) NOT NULL,
  `token` varchar(32) COLLATE utf8_bin NOT NULL,
  `owner` varchar(1024) COLLATE utf8_bin NOT NULL,
  `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `flag` tinyint(4) NOT NULL DEFAULT '0',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `t_ip_config`
--

CREATE TABLE `t_ip_config` (
  `Fip` int(10) unsigned NOT NULL,
  `FareaId` int(10) unsigned NOT NULL,
  `FcityId` int(10) unsigned NOT NULL,
  `FidcId` int(10) unsigned NOT NULL,
  `Fflag` tinyint(4) DEFAULT '0',
  `Fstamp` datetime NOT NULL,
  `Fflow` int(10) unsigned NOT NULL,
  PRIMARY KEY (`Fip`),
  KEY `idx_Fflow` (`Fflow`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

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
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

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
  PRIMARY KEY (`Fip`,`FmodId`,`FcmdId`),
  KEY `Fflow` (`Fflow`),
  KEY `idx1` (`FmodId`,`FcmdId`,`FsetId`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

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
  PRIMARY KEY (`FmodId`,`Ffrom`,`Fto`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- 表的结构 `start_lock`
--

CREATE TABLE `start_lock` (
  `lock_id` int(11) NOT NULL COMMENT '锁序号',
  `lock_key` varchar(32) COLLATE utf8_bin NOT NULL COMMENT '锁的名字',
  `server` varchar(32) COLLATE utf8_bin NOT NULL COMMENT '持有启动锁的Server',
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`lock_id`,`lock_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- 转存表中的数据 `start_lock`
--

INSERT INTO `start_lock` (`lock_id`, `lock_key`, `server`, `mtime`) VALUES
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin COMMENT='用以生成sid';

--
-- 转存表中的数据 `cl5_module`
--

insert into cl5_module(module_id, interface_id, range_num) values(3000001, 1, 0);

-- --------------------------------------------------------

--
-- 表的结构 `mesh`
--
CREATE TABLE `mesh` (
  `id`   varchar(32)  COLLATE utf8_bin NOT NULL, /*网格ID*/
  `name` varchar(128) COLLATE utf8_bin NOT NULL, /*网格名*/
  `department` varchar(1024) COLLATE utf8_bin DEFAULT NULL, /*网格所属部门*/
  `business` varchar(128) COLLATE utf8_bin NOT NULL, /*网格所属业务*/
  `managed` tinyint(4) NOT NULL, /*是否托管*/
  `istio_version` varchar(64) COLLATE utf8_bin, /*istio版本*/
  `data_cluster` varchar(1024) COLLATE utf8_bin, /*数据面集群*/
  `revision` varchar(32) COLLATE utf8_bin NOT NULL, /*规则版本号*/
  `comment` varchar(1024) COLLATE utf8_bin DEFAULT NULL, /*规则描述*/
  `token` varchar(32) COLLATE utf8_bin NOT NULL, /*规则鉴权token*/
  `owner` varchar(1024) COLLATE utf8_bin NOT NULL, /*规则的拥有者*/
  `flag` tinyint(4) NOT NULL DEFAULT '0', /*规则是否有效，0为有效，1为无效，己被删除了*/
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `name` (`name`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `mesh_service`
--
CREATE TABLE `mesh_service` (
  `id` varchar(32)  COLLATE utf8_bin NOT NULL, /*网格规则ID*/
  `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL, /*网格名*/
  `service_id` varchar(32) COLLATE utf8_bin NOT NULL, /*服务ID*/
  `namespace` varchar(64) COLLATE utf8_bin NOT NULL, /*服务命名空间*/
  `service` varchar(128) COLLATE utf8_bin NOT NULL, /*服务名*/
  `mesh_namespace` varchar(64) COLLATE utf8_bin NOT NULL, /*映射到网格的命名空间*/
  `mesh_service` varchar(128) COLLATE utf8_bin NOT NULL, /*映射到网格的服务名*/
  `location` varchar(16) COLLATE utf8_bin NOT NULL, /*服务处于网格哪个位置*/
  `export_to` varchar(1024) COLLATE utf8_bin NOT NULL, /*服务可以被哪些命名空间所见*/
  `revision` varchar(32) COLLATE utf8_bin NOT NULL, /*规则版本号*/
  `flag` tinyint(4) NOT NULL DEFAULT '0', /*规则是否有效，0为有效，1为无效，己被删除了*/
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `relation` (`mesh_id`,`mesh_namespace`,`mesh_service`),
  KEY `namespace`(`namespace`),
  KEY `service`(`service`),
  KEY `location`(`location`),
  KEY `export_to`(`export_to`),
  KEY `mtime` (`mtime`),
  KEY `flag`( `flag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `mesh_service_revision`
--
CREATE TABLE `mesh_service_revision` (
  `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL, /*网格名*/
  `revision` varchar(32) COLLATE utf8_bin NOT NULL, /*规则版本号*/
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`mesh_id`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- 表的结构 `mesh_resource`
--
CREATE TABLE `mesh_resource` (
  `id` varchar(32) COLLATE utf8_bin NOT NULL, /*网格规则ID*/
  `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL, /*网格名*/
  `name` varchar(64) COLLATE utf8_bin NOT NULL, /*规则名*/
  `mesh_namespace` varchar(64) COLLATE utf8_bin NOT NULL, /*规则所处的网格命名空间*/
  `type_url` varchar(96) COLLATE utf8_bin NOT NULL, /*规则类型，如virtualService*/
  `revision` varchar(32) COLLATE utf8_bin NOT NULL, /*规则版本号*/
  `body` text, /*规则内容，json格式字符串*/
  `flag` tinyint(4) NOT NULL DEFAULT '0', /*规则是否有效，0为有效，1为无效，己被删除了*/
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name`(`mesh_id`, `name`, `mesh_namespace`, `type_url`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- 表的结构 `mesh_revision`
--
CREATE TABLE `mesh_resource_revision` (
  `mesh_id` varchar(32) COLLATE utf8_bin NOT NULL, /*规则所属网格ID*/
  `type_url` varchar(96) COLLATE utf8_bin NOT NULL, /*规则类型，如virtualService*/
  `revision` varchar(32) COLLATE utf8_bin NOT NULL, /*规则集合的版本号，同一个网格下面所有规则集合的总体版本号*/
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`mesh_id`, `type_url`),
  KEY `mtime` (`mtime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

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
  UNIQUE KEY `unique_service` (`callee_service_id`,`caller_service_business`,`set_key`),
  KEY `mtime` (`mtime`),
  KEY `name` (`name`),
  KEY `creator` (`creator`),
  KEY `callee_service` (`callee_service_env`,`callee_service_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------
--
-- flux规则版本关联表的结构 `ratelimit_flux_rule_revision`
--
CREATE TABLE `ratelimit_flux_rule_revision` (
  `service_id` varchar(32) COLLATE utf8_bin NOT NULL,
  `last_revision` varchar(40) COLLATE utf8_bin NOT NULL,
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
-- --------------------------------------------------------
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
