/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(256) PRIMARY KEY,
    password TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE roster_items (
    username VARCHAR(256) NOT NULL,
    jid VARCHAR(512) NOT NULL,
    name TEXT NOT NULL,
    subscription TEXT NOT NULL,
    groups TEXT NOT NULL,
    ask BOOL NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (username, jid)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_roster_items_username ON roster_items(username);

CREATE TABLE roster_approval_notifications (
    username VARCHAR(256) NOT NULL,
    jid VARCHAR(256) NOT NULL,
    notification TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (username, jid)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX i_roster_approval_notifications_jid ON roster_approval_notifications(jid);

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
