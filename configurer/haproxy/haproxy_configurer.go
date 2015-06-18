package haproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router/utils"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/pivotal-golang/lager"
)

type HaProxyConfigurer struct {
	logger             lager.Logger
	baseConfigFilePath string
	configFilePath     string
	configFileLock     *sync.Mutex
}

func NewHaProxyConfigurer(logger lager.Logger, baseConfigFilePath string, configFilePath string) (*HaProxyConfigurer, error) {
	if !utils.FileExists(baseConfigFilePath) {
		return nil, errors.New(fmt.Sprintf("%s: [%s]", cf_tcp_router.ErrRouterConfigFileNotFound, baseConfigFilePath))
	}
	if !utils.FileExists(configFilePath) {
		return nil, errors.New(fmt.Sprintf("%s: [%s]", cf_tcp_router.ErrRouterConfigFileNotFound, configFilePath))
	}
	return &HaProxyConfigurer{
		logger:             logger,
		baseConfigFilePath: baseConfigFilePath,
		configFilePath:     configFilePath,
		configFileLock:     new(sync.Mutex),
	}, nil
}

func (h *HaProxyConfigurer) Configure(routingTable models.RoutingTable) error {
	h.configFileLock.Lock()
	defer h.configFileLock.Unlock()

	err := h.createConfigBackup()
	if err != nil {
		return err
	}

	cfgContent, err := ioutil.ReadFile(h.baseConfigFilePath)
	if err != nil {
		h.logger.Error("failed-reading-base-config-file", err, lager.Data{"base-config-file": h.baseConfigFilePath})
		return err
	}
	for key, entry := range routingTable.Entries {
		cfgContent, err = h.appendListenConfiguration(key, entry, cfgContent)
		if err != nil {
			return err
		}
	}

	return h.writeToConfig(cfgContent)
}

func (h *HaProxyConfigurer) appendListenConfiguration(
	key models.RoutingKey,
	entry models.RoutingTableEntry,
	cfgContent []byte) ([]byte, error) {
	var buff bytes.Buffer
	_, err := buff.Write(cfgContent)
	if err != nil {
		h.logger.Error("failed-copying-config-file", err, lager.Data{"config-file": h.configFilePath})
		return nil, err
	}

	_, err = buff.WriteString("\n")
	if err != nil {
		h.logger.Error("failed-writing-to-buffer", err)
		return nil, err
	}

	var listenCfgStr string
	listenCfgStr, err = RoutingTableEntryToHaProxyConfig(key, entry)
	if err != nil {
		h.logger.Error("failed-marshaling-routing-table-entry", err)
		return nil, err
	}

	_, err = buff.WriteString(listenCfgStr)
	if err != nil {
		h.logger.Error("failed-writing-to-buffer", err)
		return nil, err
	}
	return buff.Bytes(), nil
}

func (h *HaProxyConfigurer) createConfigBackup() error {
	h.logger.Debug("reading-config-file", lager.Data{"config-file": h.configFilePath})
	cfgContent, err := ioutil.ReadFile(h.configFilePath)
	if err != nil {
		h.logger.Error("failed-reading-base-config-file", err, lager.Data{"config-file": h.configFilePath})
		return err
	}
	backupConfigFileName := fmt.Sprintf("%s.bak", h.configFilePath)
	err = utils.WriteToFile(cfgContent, backupConfigFileName)
	if err != nil {
		h.logger.Error("failed-to-backup-config", err, lager.Data{"config-file": h.configFilePath})
		return err
	}
	return nil
}

func (h *HaProxyConfigurer) writeToConfig(cfgContent []byte) error {
	tmpConfigFileName := fmt.Sprintf("%s.tmp", h.configFilePath)
	err := utils.WriteToFile(cfgContent, tmpConfigFileName)
	if err != nil {
		h.logger.Error("failed-to-write-temp-config", err, lager.Data{"temp-config-file": tmpConfigFileName})
		return err
	}

	err = os.Rename(tmpConfigFileName, h.configFilePath)
	if err != nil {
		h.logger.Error(
			"failed-renaming-temp-config-file",
			err,
			lager.Data{"config-file": h.configFilePath, "temp-config-file": tmpConfigFileName})
		return err
	}
	return nil
}
