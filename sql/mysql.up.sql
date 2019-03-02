/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(256) PRIMARY KEY,
    password TEXT NOT NULL,
    last_presence TEXT NOT NULL,
    last_presence_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE roster_notifications (
    contact VARCHAR(256) NOT NULL,
    jid VARCHAR(512) NOT NULL,
    elements TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (contact, jid)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_roster_notifications_jid ON roster_notifications(jid);

CREATE TABLE roster_items (
    username VARCHAR(256) NOT NULL,
    jid VARCHAR(512) NOT NULL,
    name TEXT NOT NULL,
    subscription TEXT NOT NULL,
    `groups` TEXT NOT NULL,
    ask BOOL NOT NULL,
    ver INT NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (username, jid)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_roster_items_username ON roster_items(username);
CREATE INDEX i_roster_items_jid ON roster_items(jid);

CREATE TABLE roster_versions (
    username VARCHAR(256) NOT NULL,
    ver INT NOT NULL DEFAULT 0,
    last_deletion_ver INT NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS blocklist_items (
    username VARCHAR(256) NOT NULL,
    jid VARCHAR(512) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY(username, jid)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_blocklist_items_username ON blocklist_items(username);

CREATE TABLE IF NOT EXISTS private_storage (
    username VARCHAR(256) NOT NULL,
    namespace VARCHAR(512) NOT NULL,
    data MEDIUMTEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (username, namespace)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_private_storage_username ON private_storage(username);

CREATE TABLE IF NOT EXISTS vcards (
    username VARCHAR(256) PRIMARY KEY,
    vcard MEDIUMTEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS offline_messages (
    username VARCHAR(256) NOT NULL,
    data MEDIUMTEXT NOT NULL,
    created_at DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_offline_messages_username ON offline_messages(username);

CREATE TABLE IF NOT EXISTS pubsub_nodes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    host VARCHAR(512) NOT NULL,
    name TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE UNIQUE INDEX i_pubsub_nodes_jid_node ON pubsub_nodes(host(256), name(512));

CREATE TABLE pubsub_node_options (
  node_id BIGINT NOT NULL,
  name TEXT NOT NULL,
  value TEXT NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_pubsub_node_options_node_id ON pubsub_node_options(node_id);

CREATE TABLE pubsub_affiliations (
  node_id BIGINT,
  jid TEXT NOT NULL,
  affiliation TEXT NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_nodes

CREATE TABLE IF NOT EXISTS pubsub_nodes (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    host       TEXT NOT NULL,
    name       TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,

    INDEX i_pubsub_nodes_host_name (host(256), name(512))

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_node_options

CREATE TABLE IF NOT EXISTS pubsub_node_options (
    node_id BIGINT NOT NULL,
    name    TEXT NOT NULL,
    value   TEXT NOT NULL,

    INDEX i_pubsub_node_options_node_id (node_id)

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- pubsub_affiliations

CREATE TABLE IF NOT EXISTS pubsub_affiliations (
    node_id     BIGINT NOT NULL,
    jid         TEXT NOT NULL,
    affiliation TEXT NOT NULL,

    INDEX i_pubsub_affiliations_jid (jid(512)),
    UNIQUE INDEX i_pubsub_affiliations_node_id_jid (node_id, jid(512))

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
    UNIQUE INDEX i_pubsub_items_node_id_item_id (node_id, item_id(36))

) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

