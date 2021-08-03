package subscriber

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/BrobridgeOrg/gravity-exporter-stan/pkg/app"
	"github.com/BrobridgeOrg/gravity-sdk/core"
	gravity_subscriber "github.com/BrobridgeOrg/gravity-sdk/subscriber"
	gravity_state_store "github.com/BrobridgeOrg/gravity-sdk/subscriber/state_store"
	gravity_sdk_types_projection "github.com/BrobridgeOrg/gravity-sdk/types/projection"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var counter uint64 = 0

var projectionPool = sync.Pool{
	New: func() interface{} {
		return &gravity_sdk_types_projection.Projection{}
	},
}

type Subscriber struct {
	app        app.App
	stateStore *gravity_state_store.StateStore
	subscriber *gravity_subscriber.Subscriber
	ruleConfig *RuleConfig
}

func NewSubscriber(a app.App) *Subscriber {
	return &Subscriber{
		app: a,
	}
}

func (subscriber *Subscriber) processData(msg *gravity_subscriber.Message) error {
	/*
		id := atomic.AddUint64((*uint64)(&counter), 1)

		if id%100 == 0 {
			log.Info(id)
		}
	*/

	pj := projectionPool.Get().(*gravity_sdk_types_projection.Projection)
	defer projectionPool.Put(pj)

	// Parsing data
	err := gravity_sdk_types_projection.Unmarshal(msg.Event.Data, pj)
	if err != nil {
		return err
	}

	// Getting channels for specific collection
	channels, ok := subscriber.ruleConfig.Subscriptions[pj.Collection]
	if !ok {
		return err
	}

	// Convert projection to record
	payload, err := pj.ToJSON()
	if err != nil {
		return err
	}

	// Send event to each channel
	conn := subscriber.app.GetEventBus().GetSTANConnection()
	for _, channel := range channels {

		for {

			err := conn.Publish(channel, payload)
			if err == nil {
				break
			}

			log.Error(err)

			<-time.After(time.Second * 5)
		}
	}

	msg.Ack()

	return nil
}

func (subscriber *Subscriber) LoadConfigFile(filename string) (*RuleConfig, error) {

	// Open and read config file
	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// Parse config
	var config RuleConfig
	json.Unmarshal(byteValue, &config)

	return &config, nil
}

func (subscriber *Subscriber) Init() error {

	// Load rules
	ruleFile := viper.GetString("rules.subscription")

	log.WithFields(log.Fields{
		"ruleFile": ruleFile,
	}).Info("Loading rules...")

	ruleConfig, err := subscriber.LoadConfigFile(ruleFile)
	if err != nil {
		return err
	}

	subscriber.ruleConfig = ruleConfig

	// Load state
	err = subscriber.InitStateStore()
	if err != nil {
		return err
	}

	// Initializing gravity node information
	viper.SetDefault("gravity.domain", "gravity")
	domain := viper.GetString("gravity.domain")
	host := viper.GetString("gravity.host")

	log.WithFields(log.Fields{
		"host": host,
	}).Info("Initializing gravity subscriber")

	// Initializing gravity subscriber and connecting to server
	viper.SetDefault("subscriber.workerCount", 4)
	options := gravity_subscriber.NewOptions()
	options.Verbose = viper.GetBool("subscriber.verbose")
	options.Domain = domain
	options.StateStore = subscriber.stateStore
	options.WorkerCount = viper.GetInt("subscriber.workerCount")

	subscriber.subscriber = gravity_subscriber.NewSubscriber(options)
	opts := core.NewOptions()
	err = subscriber.subscriber.Connect(host, opts)
	if err != nil {
		return err
	}

	// Setup data handler
	subscriber.subscriber.SetEventHandler(subscriber.eventHandler)

	// Register subscriber
	log.Info("Registering subscriber")
	subscriberID := viper.GetString("subscriber.subscriberID")
	subscriberName := viper.GetString("subscriber.subscriberName")
	err = subscriber.subscriber.Register(gravity_subscriber.SubscriberType_Exporter, "stan", subscriberID, subscriberName)
	if err != nil {
		return err
	}

	// Subscribe to collections
	err = subscriber.subscriber.SubscribeToCollections(subscriber.ruleConfig.Subscriptions)
	if err != nil {
		return err
	}

	// Subscribe to pipelines
	err = subscriber.initializePipelines()
	if err != nil {
		return err
	}

	return nil
}

func (subscriber *Subscriber) initializePipelines() error {

	// Subscribe to pipelines
	log.WithFields(log.Fields{}).Info("Subscribing to gravity pipelines...")
	viper.SetDefault("subscriber.pipelineStart", 0)
	viper.SetDefault("subscriber.pipelineEnd", -1)

	pipelineStart := viper.GetInt64("subscriber.pipelineStart")
	pipelineEnd := viper.GetInt64("subscriber.pipelineEnd")

	// Subscribe to all pipelines
	if pipelineStart == 0 && pipelineEnd == -1 {
		err := subscriber.subscriber.AddAllPipelines()
		if err != nil {
			return err
		}

		return nil
	}

	// Subscribe to pipelines in then range
	if pipelineStart < 0 {
		return fmt.Errorf("subscriber.pipelineStart should be higher than -1")
	}

	if pipelineStart > pipelineEnd {
		if pipelineEnd != -1 {
			return fmt.Errorf("subscriber.pipelineStart should be less than subscriber.pipelineEnd")
		}
	}

	count, err := subscriber.subscriber.GetPipelineCount()
	if err != nil {
		return err
	}

	if pipelineEnd == -1 {
		pipelineEnd = int64(count) - 1
	}

	pipelines := make([]uint64, 0, pipelineEnd-pipelineStart)
	for i := pipelineStart; i <= pipelineEnd; i++ {
		pipelines = append(pipelines, uint64(i))
	}

	err = subscriber.subscriber.SubscribeToPipelines(pipelines)
	if err != nil {
		return err
	}

	return nil
}

func (subscriber *Subscriber) eventHandler(msg *gravity_subscriber.Message) {

	err := subscriber.processData(msg)
	if err != nil {
		log.Error(err)
		return
	}
}

func (subscriber *Subscriber) Run() error {

	log.WithFields(log.Fields{}).Info("Starting to fetch data from gravity...")

	subscriber.subscriber.Start()

	return nil
}
