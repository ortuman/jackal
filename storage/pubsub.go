/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

type pubSubStorage interface {
	UpsertPubSubNode(node *pubsubmodel.Node) error
	FetchPubSubNode(host, name string) (*pubsubmodel.Node, error)
	DeletePubSubNode(host, name string) error

	PubSubNodeExists(host, name string) (bool, error)

	UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error
	FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error)

	UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error
	FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error)
}

func UpsertPubSubNode(node *pubsubmodel.Node) error {
	return inst.UpsertPubSubNode(node)
}

func FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	return inst.FetchPubSubNode(host, name)
}

func DeletePubSubNode(host, name string) error {
	return inst.DeletePubSubNode(host, name)
}

func PubSubNodeExists(host, name string) (bool, error) {
	return inst.PubSubNodeExists(host, name)
}

func UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return inst.UpsertPubSubNodeItem(item, host, name, maxNodeItems)
}

func FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	return inst.FetchPubSubNodeItems(host, name)
}

func UpsertPubSubNodeAffiliation(affiliatiaon *pubsubmodel.Affiliation, host, name string) error {
	return inst.UpsertPubSubNodeAffiliation(affiliatiaon, host, name)
}

func FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	return inst.FetchPubSubNodeAffiliations(host, name)
}
