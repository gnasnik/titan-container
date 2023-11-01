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
    created_at DATETIME     DEFAULT NULL,
    updated_at DATETIME     DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY `uniq_deployment_id_image` (`deployment_id`, `image`)
    )ENGINE=InnoDB COMMENT='services';