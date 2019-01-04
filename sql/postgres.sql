/*
 * Copyright (c) 2018 robzon.
 * See the LICENSE file for more information.
 */

CREATE TABLE IF NOT EXISTS users (
    username            VARCHAR(256) PRIMARY KEY,
    password            TEXT NOT NULL,
    last_presence       TEXT NOT NULL,
    last_presence_at    TIMESTAMP NOT NULL,
    updated_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS roster_notifications (
    contact     VARCHAR(256) NOT NULL,
    jid         VARCHAR(512) NOT NULL,
    elements    TEXT NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP NOT NULL,

    PRIMARY KEY (contact, jid)
);

CREATE INDEX IF NOT EXISTS i_roster_notifications_jid ON roster_notifications(jid);

CREATE TABLE IF NOT EXISTS roster_items (
    username        VARCHAR(256) NOT NULL,
    jid             VARCHAR(512) NOT NULL,
    name            TEXT NOT NULL,
    subscription    TEXT NOT NULL,
    groups          TEXT NOT NULL,
    ask BOOL        NOT NULL,
    ver             INT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMP NOT NULL,
    created_at      TIMESTAMP NOT NULL,
    
    PRIMARY KEY (username, jid)
);

CREATE INDEX IF NOT EXISTS i_roster_items_username ON roster_items(username);
CREATE INDEX IF NOT EXISTS i_roster_items_jid ON roster_items(jid);

CREATE TABLE IF NOT EXISTS roster_versions (
    username            VARCHAR(256) NOT NULL,
    ver                 INT NOT NULL DEFAULT 0,
    last_deletion_ver   INT NOT NULL DEFAULT 0,
    updated_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL,
    
    PRIMARY KEY (username)
);

CREATE TABLE IF NOT EXISTS blocklist_items (
    username        VARCHAR(256) NOT NULL,
    jid             VARCHAR(512) NOT NULL,
    created_at      TIMESTAMP NOT NULL,
    
    PRIMARY KEY(username, jid)
);

CREATE INDEX IF NOT EXISTS i_blocklist_items_username ON blocklist_items(username);

CREATE TABLE IF NOT EXISTS private_storage (
    username        VARCHAR(256) NOT NULL,
    namespace       VARCHAR(512) NOT NULL,
    data            TEXT NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    created_at      TIMESTAMP NOT NULL,
    
    PRIMARY KEY (username, namespace)
);

CREATE INDEX IF NOT EXISTS i_private_storage_username ON private_storage(username);

CREATE TABLE IF NOT EXISTS vcards (
    username        VARCHAR(256) PRIMARY KEY,
    vcard           TEXT NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    created_at      TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS offline_messages (
    username        VARCHAR(256) NOT NULL,
    data            TEXT NOT NULL,
    created_at      TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS i_offline_messages_username ON offline_messages(username);
