CREATE TABLE IF NOT EXISTS deployments(
    id VARCHAR(128) NOT NULL UNIQUE,
    owner VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL DEFAULT '',
    state INT DEFAULT 0,
    type INT DEFAULT 0,
    authority TINYINT(1) DEFAULT 0,
    version VARCHAR(128) DEFAULT '',
    balance FLOAT        DEFAULT 0,
    cost FLOAT        DEFAULT 0,
    provider_id VARCHAR(128) NOT NULL,
    expiration DATETIME     DEFAULT NULL,
    created_at DATETIME     DEFAULT NULL,
    updated_at DATETIME     DEFAULT NULL
    )ENGINE=InnoDB COMMENT='deployments';

CREATE TABLE IF NOT EXISTS providers(
    id VARCHAR(128) NOT NULL UNIQUE,
    owner VARCHAR(128) NOT NULL,
    host_uri VARCHAR(128) NOT NULL,
    ip VARCHAR(128) NOT NULL,
    state INT DEFAULT 0,
    created_at DATETIME     DEFAULT NULL,
    updated_at DATETIME     DEFAULT NULL,
    PRIMARY KEY (id)
    )ENGINE=InnoDB COMMENT='providers';

CREATE TABLE IF NOT EXISTS properties(
    id INT UNSIGNED AUTO_INCREMENT,
    provider_id VARCHAR(128) NOT NULL UNIQUE,
    app_id VARCHAR(128) NOT NULL,
    app_type INT DEFAULT 0,
    created_at DATETIME     DEFAULT NULL,
    updated_at DATETIME     DEFAULT NULL,
    PRIMARY KEY (id),
    KEY idx_provider_id (provider_id)
    )ENGINE=InnoDB COMMENT='properties';


CREATE TABLE IF NOT EXISTS services(
    id INT UNSIGNED AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL DEFAULT '',
    image VARCHAR(128) NOT NULL DEFAULT '',
    ports VARCHAR(256) NOT NULL DEFAULT '',
    state INT DEFAULT 0,
    cpu FLOAT        DEFAULT 0,
    gpu FLOAT        DEFAULT 0,
    memory FLOAT        DEFAULT 0,
    storage VARCHAR(128),
    env VARCHAR(128) DEFAULT NULL,
    arguments VARCHAR(128) DEFAULT NULL,
    deployment_id VARCHAR(128) NOT NULL,
    error_message VARCHAR(128) DEFAULT NULL,
    replicas INT NOT NULL DEFAULT 0,
    created_at DATETIME     DEFAULT NULL,
    updated_at DATETIME     DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY `uniq_deployment_id_image` (`deployment_id`, `image`)
    )ENGINE=InnoDB COMMENT='services';


CREATE TABLE IF NOT EXISTS domains(
    id INT UNSIGNED AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL DEFAULT '',
    deployment_id VARCHAR(128) NOT NULL DEFAULT '',
    provider_id VARCHAR(128) NOT NULL DEFAULT '',
    state VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME     DEFAULT NULL,
    updated_at DATETIME     DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_name (name)
    )ENGINE=InnoDB COMMENT='domains'
