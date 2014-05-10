package common

import (
	"github.com/gocql/gocql"
	"log"
	"strings"
	"time"
)

type CassandraManager struct {
	cluster *gocql.ClusterConfig
	session *gocql.Session
}

/*
CREATE KEYSPACE preview
  WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 };
USE preview;
CREATE TABLE IF NOT EXISTS generated_assets (id varchar, source varchar, status varchar, template_id varchar, message blob, PRIMARY KEY (id));
CREATE TABLE IF NOT EXISTS active_generated_assets (id varchar PRIMARY KEY);
CREATE TABLE IF NOT EXISTS waiting_generated_assets (id varchar, source varchar, template varchar, state varchar, PRIMARY KEY(template, source, id));
CREATE INDEX IF NOT EXISTS ON generated_assets (source);
CREATE INDEX IF NOT EXISTS ON generated_assets (status);
CREATE INDEX IF NOT EXISTS ON generated_assets (template_id);
CREATE TABLE IF NOT EXISTS source_assets (id varchar, type varchar, message blob, PRIMARY KEY (id, type));
CREATE INDEX IF NOT EXISTS ON source_assets (type);

TRUNCATE preview.source_assets;
TRUNCATE preview.generated_assets;
TRUNCATE preview.active_generated_assets;
TRUNCATE preview.waiting_generated_assets;

*/

type cassandraSourceAssetStorageManager struct {
	cassandraManager *CassandraManager
	nodeId           string
	keyspace         string
}

type cassandraGeneratedAssetStorageManager struct {
	cassandraManager *CassandraManager
	templateManager  TemplateManager
	nodeId           string
	keyspace         string
}

func NewCassandraManager(hosts []string, keyspace string) (*CassandraManager, error) {
	cm := new(CassandraManager)

	cm.cluster = gocql.NewCluster(hosts...)
	cm.cluster.Consistency = gocql.Any
	cm.cluster.Keyspace = keyspace

	session, err := cm.cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	cm.session = session

	return cm, nil
}

func NewCassandraSourceAssetStorageManager(cm *CassandraManager, nodeId, keyspace string) (SourceAssetStorageManager, error) {
	csasm := new(cassandraSourceAssetStorageManager)
	csasm.cassandraManager = cm
	csasm.nodeId = nodeId
	csasm.keyspace = keyspace
	return csasm, nil
}

func NewCassandraGeneratedAssetStorageManager(cm *CassandraManager, templateManager TemplateManager, nodeId, keyspace string) (GeneratedAssetStorageManager, error) {
	cgasm := new(cassandraGeneratedAssetStorageManager)
	cgasm.cassandraManager = cm
	cgasm.templateManager = templateManager
	cgasm.nodeId = nodeId
	cgasm.keyspace = keyspace
	return cgasm, nil
}

func (cm *CassandraManager) Session() *gocql.Session {
	return cm.session
}

func (cm *CassandraManager) Stop() {
	if !cm.session.Closed() {
		cm.session.Close()
	}
}

func (sasm *cassandraSourceAssetStorageManager) Store(sourceAsset *SourceAsset) error {
	sourceAsset.CreatedBy = sasm.nodeId
	sourceAsset.UpdatedBy = sasm.nodeId
	payload, err := sourceAsset.Serialize()
	if err != nil {
		log.Println("Error serializing source asset:", err)
		return err
	}
	err = sasm.cassandraManager.Session().Query(`INSERT INTO `+sasm.keyspace+`.source_assets (id, type, message) VALUES (?, ?, ?)`, sourceAsset.Id, sourceAsset.IdType, payload).Exec()
	if err != nil {
		log.Println("Error persisting source asset:", err)
		return err
	}

	return nil
}

func (sasm *cassandraSourceAssetStorageManager) FindBySourceAssetId(id string) ([]*SourceAsset, error) {
	results := make([]*SourceAsset, 0, 0)

	iter := sasm.cassandraManager.Session().Query(`SELECT id, message FROM `+sasm.keyspace+`.source_assets WHERE id = ?`, id).Consistency(gocql.One).Iter()
	var sourceAssetId string
	var message []byte
	for iter.Scan(&sourceAssetId, &message) {
		sourceAsset, err := newSourceAssetFromJson(message)
		if err != nil {
			return nil, err
		}
		results = append(results, sourceAsset)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

func (gasm *cassandraGeneratedAssetStorageManager) Store(generatedAsset *GeneratedAsset) error {
	generatedAsset.CreatedBy = gasm.nodeId
	generatedAsset.UpdatedBy = gasm.nodeId
	payload, err := generatedAsset.Serialize()
	if err != nil {
		log.Println("Error serializing source asset:", err)
		return err
	}

	log.Println("Storing generated asset", generatedAsset)

	batch := gasm.cassandraManager.Session().NewBatch(gocql.UnloggedBatch)
	batch.Query(`INSERT INTO `+gasm.keyspace+`.generated_assets (id, source, status, template_id, message) VALUES (?, ?, ?, ?, ?)`,
		generatedAsset.Id, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType, generatedAsset.Status, generatedAsset.TemplateId, payload)

	if generatedAsset.Status == GeneratedAssetStatusWaiting {
		templateGroup, err := gasm.templateGroup(generatedAsset.TemplateId)
		if err != nil {
			return err
		}
		batch.Query(`INSERT INTO `+gasm.keyspace+`.waiting_generated_assets (id, source, template) VALUES (?, ?, ?)`,
			generatedAsset.Id, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType, templateGroup)
	}

	err = gasm.cassandraManager.Session().ExecuteBatch(batch)
	if err != nil {
		log.Println("Error executing batch:", err)
		return err
	}

	return nil
}

func (gasm *cassandraGeneratedAssetStorageManager) templateGroup(id string) (string, error) {
	templates, err := gasm.templateManager.FindByIds([]string{id})
	if err != nil {
		return "", err
	}
	if len(templates) != 1 {
		return "", ErrorNoTemplateForId
	}
	template := templates[0]
	return template.Group, nil
}

func (gasm *cassandraGeneratedAssetStorageManager) Update(generatedAsset *GeneratedAsset) error {
	generatedAsset.UpdatedAt = time.Now().UnixNano()
	generatedAsset.UpdatedBy = gasm.nodeId
	payload, err := generatedAsset.Serialize()
	if err != nil {
		log.Println("Error serializing generated asset:", err)
		return err
	}
	batch := gasm.cassandraManager.Session().NewBatch(gocql.UnloggedBatch)
	batch.Query(`UPDATE `+gasm.keyspace+`.generated_assets SET status = ?, message = ? WHERE id = ?`, generatedAsset.Status, payload, generatedAsset.Id)

	if generatedAsset.Status == GeneratedAssetStatusScheduled || generatedAsset.Status == GeneratedAssetStatusProcessing {
		templateGroup, err := gasm.templateGroup(generatedAsset.TemplateId)
		if err != nil {
			return err
		}
		batch.Query(`DELETE FROM `+gasm.keyspace+`.waiting_generated_assets WHERE id = ? AND template = ? AND source = ?`, generatedAsset.Id, templateGroup, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType)
		batch.Query(`INSERT INTO `+gasm.keyspace+`.active_generated_assets (id) VALUES (?)`, generatedAsset.Id)
	}
	if generatedAsset.Status == GeneratedAssetStatusComplete || strings.HasPrefix(generatedAsset.Status, GeneratedAssetStatusFailed) {
		batch.Query(`DELETE FROM `+gasm.keyspace+`.active_generated_assets WHERE id = ?`, generatedAsset.Id)
	}
	err = gasm.cassandraManager.Session().ExecuteBatch(batch)
	if err != nil {
		log.Println("Error executing batch:", err)
		return err
	}
	return nil
}

func (gasm *cassandraGeneratedAssetStorageManager) FindById(id string) (*GeneratedAsset, error) {
	generatedAssets, err := gasm.getIds([]string{id})
	if err != nil {
		return nil, err
	}
	if len(generatedAssets) == 0 {
		return nil, ErrorNoGeneratedAssetsFoundForId
	}
	return generatedAssets[0], nil
}

func (gasm *cassandraGeneratedAssetStorageManager) FindByIds(ids []string) ([]*GeneratedAsset, error) {
	return gasm.getIds(ids)
}

func (gasm *cassandraGeneratedAssetStorageManager) FindBySourceAssetId(id string) ([]*GeneratedAsset, error) {
	results := make([]*GeneratedAsset, 0, 0)

	iter := gasm.cassandraManager.Session().Query(`SELECT id, message FROM `+gasm.keyspace+`.generated_assets WHERE source = ?`, id+"origin").Consistency(gocql.One).Iter()
	var generatedAssetId string
	var message []byte
	for iter.Scan(&generatedAssetId, &message) {
		generatedAsset, err := newGeneratedAssetFromJson(message)
		if err != nil {
			return nil, err
		}
		results = append(results, generatedAsset)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

func (gasm *cassandraGeneratedAssetStorageManager) FindWorkForService(serviceName string, workCount int) ([]*GeneratedAsset, error) {
	templates, err := gasm.templateManager.FindByRenderService(serviceName)
	if err != nil {
		return nil, err
	}
	generatedAssetIds, err := gasm.getWaitingAssets(templates[0].Group, workCount)
	if err != nil {
		return nil, err
	}

	return gasm.getIds(generatedAssetIds)
}

func (gasm *cassandraGeneratedAssetStorageManager) getWaitingAssets(group string, count int) ([]string, error) {
	results := make([]string, 0, 0)

	iter := gasm.cassandraManager.Session().Query(`SELECT id FROM `+gasm.keyspace+`.waiting_generated_assets WHERE template = ? LIMIT ?`, group, count).Consistency(gocql.One).Iter()
	var generatedAssetId string
	for iter.Scan(&generatedAssetId) {
		results = append(results, generatedAssetId)
		log.Println("waiting_generated_assets from cassandra", generatedAssetId)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

func (gasm *cassandraGeneratedAssetStorageManager) getIds(ids []string) ([]*GeneratedAsset, error) {
	results := make([]*GeneratedAsset, 0, 0)

	args := make([]interface{}, len(ids))
	for i, v := range ids {
		args[i] = interface{}(v)
	}

	iter := gasm.cassandraManager.Session().Query(`SELECT message FROM `+gasm.keyspace+`.generated_assets WHERE id in (`+buildIn(len(ids))+`)`, args...).Consistency(gocql.One).Iter()
	var message []byte
	for iter.Scan(&message) {
		generatedAsset, err := newGeneratedAssetFromJson(message)
		if err != nil {
			return nil, err
		}
		results = append(results, generatedAsset)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return results, nil
}
