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

/* 配置分组表变更操作 */
ALTER TABLE `config_file_group`
    ADD COLUMN `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否被删除';

ALTER TABLE `config_file_group`
    ADD COLUMN `business` varchar(64) DEFAULT NULL comment 'Service business information';

ALTER TABLE `config_file_group`
    ADD COLUMN `department` varchar(1024) DEFAULT NULL comment 'Service department information';

ALTER TABLE `config_file_group`
    ADD COLUMN `metadata` text COMMENT '配置分组标签';

/* 配置发布表变更操作 */
ALTER TABLE `config_file_release`
    ADD COLUMN `tags` text COMMENT '文件标签';

ALTER TABLE `config_file_release`
ADD COLUMN `active` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否处于使用中';

ALTER TABLE `config_file_release`
ADD COLUMN `format` varchar(16) DEFAULT 'text' COMMENT '文件格式，枚举值';

ALTER TABLE `config_file_release`
    ADD COLUMN `description` varchar(512) DEFAULT '' COMMENT '发布描述';

ALTER TABLE `config_file_release`
    ADD UNIQUE KEY `uk_file_release` (`namespace`, `group`, `file_name`, `name`);

ALTER TABLE `config_file_release`
    DROP KEY `uk_file`;

/* 配置历史表变更操作 */
ALTER TABLE `config_file_release_history`
    ADD COLUMN `reason` varchar(3000) DEFAULT '' COMMENT '发布原因';

ALTER TABLE `config_file_release_history`
    MODIFY COLUMN `tags` text COMMENT '文件标签';

ALTER TABLE `config_file_release_history`
    ADD COLUMN `version` bigint(11) COMMENT '版本号, 每次发布自增1';

ALTER TABLE `config_file_release_history`
    ADD COLUMN `description` varchar(512) DEFAULT NULL COMMENT '发布描述';