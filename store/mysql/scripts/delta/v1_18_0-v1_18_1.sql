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

-- 服务可见性
ALTER TABLE service_contract
ADD COLUMN `type` VARCHAR(128) NOT NULL COMMENT '服务契约接口名称';

ALTER TABLE service_contract_detail
ADD COLUMN `namespace` VARCHAR(64)  NOT NULL COMMENT '命名空间';

ALTER TABLE service_contract_detail
ADD COLUMN `service`   VARCHAR(128) NOT NULL COMMENT '服务名称';

ALTER TABLE service_contract_detail
ADD COLUMN `protocol`  VARCHAR(32)  NOT NULL COMMENT '当前契约对应的协议信息 e.g. http/dubbo/grpc/thrift';

ALTER TABLE service_contract_detail
ADD COLUMN `version`   VARCHAR(64)  NOT NULL COMMENT '服务契约版本';

ALTER TABLE service_contract_detail
ADD COLUMN `type` VARCHAR(128) NOT NULL COMMENT '服务契约接口名称';

CREATE TABLE lane_group
(
    id       varchar(128) not null comment '泳道分组 ID',
    name     varchar(64)  not null comment '泳道分组名称',
    rule     text         not null comment '规则的 json 字符串',
    description varchar(3000) comment '规则描述',
    revision VARCHAR(40)  NOT NULL comment '规则摘要',
    flag     tinyint               default 0 comment '软删除标识位',
    ctime    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    mtime    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`)
) ENGINE = InnoDB;

CREATE TABLE lane_rule
(
    id          varchar(128) not null comment '规则 id',
    name        varchar(64)  not null comment '规则名称',
    group_name  varchar(64)  not null comment '泳道分组名称',
    rule        text         not null comment '规则的 json 字符串',
    revision    VARCHAR(40)  NOT NULL comment '规则摘要',
    description varchar(3000) comment '规则描述',
    enable      tinyint comment '是否启用',
    flag        tinyint               default 0 comment '软删除标识位',
    priority    bigint       NOT NULL DEFAULT 0 comment '泳道规则优先级',
    ctime       timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    etime       timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    mtime       timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`group_name`, `name`)
) ENGINE = InnoDB;
