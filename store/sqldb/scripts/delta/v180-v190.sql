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

CREATE TABLE `config_file_template` (
    `id` bigint(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name` varchar(128) COLLATE utf8_bin NOT NULL COMMENT '配置文件模板名称',
    `content` longtext COLLATE utf8_bin NOT NULL COMMENT '配置文件模板内容',
    `format` varchar(16) COLLATE utf8_bin DEFAULT 'text' COMMENT '模板文件格式',
    `comment` varchar(512) COLLATE utf8_bin DEFAULT NULL COMMENT '模板描述信息',
    `flag` tinyint(4) NOT NULL DEFAULT '0' COMMENT '软删除标记位',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by` varchar(32) COLLATE utf8_bin DEFAULT NULL COMMENT '创建人',
    `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `modify_by` varchar(32) COLLATE utf8_bin DEFAULT NULL COMMENT '最后更新人',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='配置文件模板表';
