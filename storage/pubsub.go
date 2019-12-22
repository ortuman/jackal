/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

type pubSubStorage interface {
	FetchHosts() (hosts []string, err error)

	UpsertNode(node *pubsubmodel.Node) error
	FetchNode(host, name string) (*pubsubmodel.Node, error)
	FetchNodes(host string) ([]pubsubmodel.Node, error)
	FetchSubscribedNodes(jid string) ([]pubsubmodel.Node, error)
	DeleteNode(host, name string) error

	UpsertNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error
	FetchNodeItems(host, name string) ([]pubsubmodel.Item, error)
	FetchNodeItemsWithIDs(host, name string, identifiers []string) ([]pubsubmodel.Item, error)
	FetchNodeLastItem(host, name string) (*pubsubmodel.Item, error)

	UpsertNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error
	FetchNodeAffiliation(host, name, jid string) (*pubsubmodel.Affiliation, error)
	FetchNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error)
	DeleteNodeAffiliation(jid, host, name string) error

	UpsertNodeSubscription(subscription *pubsubmodel.Subscription, host, name string) error
	FetchNodeSubscriptions(host, name string) ([]pubsubmodel.Subscription, error)
	DeleteNodeSubscription(jid, host, name string) error
}

func FetchHosts() (hosts []string, err error) {
	return inst.FetchHosts()
}

func UpsertNode(node *pubsubmodel.Node) error {
	return inst.UpsertNode(node)
}

func FetchNode(host, name string) (*pubsubmodel.Node, error) {
	return inst.FetchNode(host, name)
}

func FetchNodes(host string) ([]pubsubmodel.Node, error) {
	return inst.FetchNodes(host)
}

func FetchSubscribedNodes(jid string) ([]pubsubmodel.Node, error) {
	return inst.FetchSubscribedNodes(jid)
}

func DeleteNode(host, name string) error {
	return inst.DeleteNode(host, name)
}

func UpsertNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return inst.UpsertNodeItem(item, host, name, maxNodeItems)
}

func FetchNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	return inst.FetchNodeItems(host, name)
}

func FetchNodeItemsWithIDs(host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
	return inst.FetchNodeItemsWithIDs(host, name, identifiers)
}

func FetchNodeLastItem(host, name string) (*pubsubmodel.Item, error) {
	return inst.FetchNodeLastItem(host, name)
}

func UpsertNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return inst.UpsertNodeAffiliation(affiliation, host, name)
}

func DeleteNodeAffiliation(jid, host, name string) error {
	return inst.DeleteNodeAffiliation(jid, host, name)
}

func FetchNodeAffiliation(host, name, jid string) (*pubsubmodel.Affiliation, error) {
	return inst.FetchNodeAffiliation(host, name, jid)
}

func FetchNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	return inst.FetchNodeAffiliations(host, name)
}

func UpsertNodeSubscription(subscription *pubsubmodel.Subscription, host, name string) error {
	return inst.UpsertNodeSubscription(subscription, host, name)
}

func FetchNodeSubscriptions(host, name string) ([]pubsubmodel.Subscription, error) {
	return inst.FetchNodeSubscriptions(host, name)
}

func DeleteNodeSubscription(jid, host, name string) error {
	return inst.DeleteNodeSubscription(jid, host, name)
}
