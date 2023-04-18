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

ALTER TABLE `ratelimit_config` ADD COLUMN `name` varchar(64) NOT NULL;
ALTER TABLE `ratelimit_config` ADD COLUMN `disable` tinyint(4)  NOT NULL DEFAULT '0';
ALTER TABLE `ratelimit_config` ADD COLUMN `etime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE `ratelimit_config` ADD COLUMN `method` varchar(512)   NOT NULL;
ALTER TABLE `ratelimit_config` MODIFY COLUMN `cluster_id` varchar(32) DEFAULT '' comment 'Cluster ID, no use';


CREATE TABLE `config_file_template` (
    `id` bigint(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name` varchar(128) COLLATE utf8_bin NOT NULL COMMENT '配置文件模板名称',
    `content` longtext COLLATE utf8_bin NOT NULL COMMENT '配置文件模板内容',
    `format` varchar(16) COLLATE utf8_bin DEFAULT 'text' COMMENT '模板文件格式',
    `comment` varchar(512) COLLATE utf8_bin DEFAULT NULL COMMENT '模板描述信息',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by` varchar(32) COLLATE utf8_bin DEFAULT NULL COMMENT '创建人',
    `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by` varchar(32) COLLATE utf8_bin DEFAULT NULL COMMENT '最后更新人',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB COMMENT='配置文件模板表';

INSERT INTO `config_file_template` (`id`,`name`,`content`,`format`,`comment`,`create_time`,`create_by`,`modify_time`,`modify_by`) VALUES (2,'spring-cloud-gateway-braining','{\n \"rules\":[\n {\n \"conditions\":[\n {\n \"key\":\"${http.query.uid}\",\n \"values\":[\n \"10000\"\n ],\n \"operation\":\"EQUALS\"\n }\n ],\n \"labels\":[\n {\n \"key\":\"env\",\n \"value\":\"green\"\n }\n ]\n }\n ]\n}','json','Spring Cloud Gateway 染色规则','2022-08-18 10:54:46','polaris','2022-08-18 10:55:22','polaris');

