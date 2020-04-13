/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

-- users

CREATE TABLE IF NOT EXISTS users (
    username              VARCHAR(256) PRIMARY KEY,
    password_scram_sha1   BLOB NOT NULL,
    password_scram_sha256 BLOB NOT NULL,
    salt                  BLOB NOT NULL,
    iteration_count       INTEGER NOT NULL,
    last_presence         TEXT NOT NULL,
    last_presence_at      DATETIME NOT NULL,
    updated_at            DATETIME NOT NULL,
    created_at            DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- presences

CREATE TABLE IF NOT EXISTS presences (
    username      VARCHAR(256) NOT NULL,
    domain        VARCHAR(256) NOT NULL,
    resource      VARCHAR(256) NOT NULL,
    presence      TEXT NOT NULL,
    node          VARCHAR(256) NOT NULL,
    ver           VARCHAR(256) NOT NULL,
    allocation_id VARCHAR(256) NOT NULL,
    updated_at    DATETIME NOT NULL,
    created_at    DATETIME NOT NULL,

    PRIMARY KEY (username, domain, resource),

    INDEX i_presences_username_domain(username, domain),
    INDEX i_presences_domain_resource(domain, resource),
    INDEX i_presences_allocation_id(allocation_id)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- capabilities

CREATE TABLE IF NOT EXISTS capabilities (
    node       VARCHAR(256) NOT NULL,
    ver        VARCHAR(256) NOT NULL,
    features   TEXT,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    PRIMARY KEY (node, ver)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- roster_notifications

CREATE TABLE IF NOT EXISTS roster_notifications (
    contact    VARCHAR(256) NOT NULL,
    jid        VARCHAR(512) NOT NULL,
    elements   TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    PRIMARY KEY (contact, jid),

    INDEX i_roster_notifications_jid (jid)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- roster_items

CREATE TABLE IF NOT EXISTS roster_items (
    username     VARCHAR(256) NOT NULL,
    jid          VARCHAR(512) NOT NULL,
    name         TEXT NOT NULL,
    subscription TEXT NOT NULL,
    `groups`     TEXT NOT NULL,
    ask          BOOL NOT NULL,
    ver          INT NOT NULL DEFAULT 0,
    updated_at   DATETIME NOT NULL,
    created_at   DATETIME NOT NULL,

    PRIMARY KEY (username, jid),

    INDEX i_roster_items_username(username),
    INDEX i_roster_items_jid     (jid)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- roster_groups

CREATE TABLE IF NOT EXISTS roster_groups (
    username     VARCHAR(256) NOT NULL,
    jid          VARCHAR(512) NOT NULL,
    `group`      TEXT NOT NULL,
    updated_at   DATETIME NOT NULL,
    created_at   DATETIME NOT NULL,

    INDEX i_roster_groups_username_jid (username, jid)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- roster_versions

CREATE TABLE IF NOT EXISTS roster_versions (
    username          VARCHAR(256) NOT NULL,
    ver               INT NOT NULL DEFAULT 0,
    last_deletion_ver INT NOT NULL DEFAULT 0,
    updated_at        DATETIME NOT NULL,
    created_at        DATETIME NOT NULL,
    PRIMARY KEY (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- blocklist_items

CREATE TABLE IF NOT EXISTS blocklist_items (
    username   VARCHAR(256) NOT NULL,
    jid        VARCHAR(512) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY(username, jid),

    INDEX i_blocklist_items_username (username)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- private_storage

CREATE TABLE IF NOT EXISTS private_storage (
    username   VARCHAR(256) NOT NULL,
    namespace  VARCHAR(512) NOT NULL,
    data       MEDIUMTEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (username, namespace),

    INDEX i_private_storage_username (username)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- vcards

CREATE TABLE IF NOT EXISTS vcards (
    username   VARCHAR(256) PRIMARY KEY,
    vcard      MEDIUMTEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- offline_messages

CREATE TABLE IF NOT EXISTS offline_messages (
    username   VARCHAR(256) NOT NULL,
    data       MEDIUMTEXT NOT NULL,
    created_at DATETIME NOT NULL,

    INDEX i_offline_messages_username (username)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_nodes

CREATE TABLE IF NOT EXISTS pubsub_nodes (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    host       TEXT NOT NULL,
    name       TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    INDEX i_pubsub_nodes_host (host(256)),
    UNIQUE INDEX i_pubsub_nodes_host_name (host(256), name(512))

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_node_options

CREATE TABLE IF NOT EXISTS pubsub_node_options (
    node_id BIGINT NOT NULL,
    name    TEXT NOT NULL,
    value   TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    INDEX i_pubsub_node_options_node_id (node_id)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_affiliations

CREATE TABLE IF NOT EXISTS pubsub_affiliations (
    node_id     BIGINT NOT NULL,
    jid         TEXT NOT NULL,
    affiliation TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    INDEX i_pubsub_affiliations_jid (jid(512)),
    UNIQUE INDEX i_pubsub_affiliations_node_id_jid (node_id, jid(512))

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_subscriptions

CREATE TABLE IF NOT EXISTS pubsub_subscriptions (
    node_id      BIGINT NOT NULL,
    subid        TEXT NOT NULL,
    jid          TEXT NOT NULL,
    subscription TEXT NOT NULL,
    updated_at   DATETIME NOT NULL,
    created_at   DATETIME NOT NULL,

    INDEX i_pubsub_subscriptions_jid (jid(512)),
    UNIQUE INDEX i_pubsub_subscriptions_node_id_jid (node_id, jid(512))

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_items

CREATE TABLE IF NOT EXISTS pubsub_items (
    node_id    BIGINT NOT NULL,
    item_id    TEXT NOT NULL,
    payload    TEXT NOT NULL,
    publisher  TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    INDEX i_pubsub_items_item_id (item_id(36)),
    INDEX i_pubsub_items_node_id_created_at (node_id, created_at),
    UNIQUE INDEX i_pubsub_items_node_id_item_id (node_id, item_id(36))

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
