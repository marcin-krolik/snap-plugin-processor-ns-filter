/*
http://www.apache.org/licenses/LICENSE-2.0.txt

Copyright 2016 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package processor

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"regexp"

	log "github.com/Sirupsen/logrus"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"
)

const (
	// name of the processor plugin
	pluginName = "ns-filter"
	// version of the processor plugin
	pluginVersion = 1
	// type of the plugin
	pluginType = plugin.ProcessorPluginType
	// configuration error
	errConfigurationNotProvided = "Required configuration item {%s} is not provided"
	// regular experion compilation error
	errRegexpCompilationError = "Could not compile provided expression {%s}, %s"
)

// New returns instance of MesosProcessor plugin
func New() *Processor {
	return &Processor{logger: log.New()}
}

// Meta returns a plugin meta data
func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(pluginName, pluginVersion, pluginType, []string{plugin.SnapGOBContentType}, []string{plugin.SnapGOBContentType})
}

// GetConfigPolicy returns config policy for MesosProcessor plugin
func (p *Processor) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	cp := cpolicy.New()
	rule1, _ := cpolicy.NewStringRule("expression", true)
	rule2, _ := cpolicy.NewStringRule("tag", true)
	pnode := cpolicy.NewPolicyNode()
	pnode.Add(rule1)
	pnode.Add(rule2)
	cp.Add([]string{"/"}, pnode)
	return cp, nil
}

// Process calculates additional derived metrics based on raw metrics received
func (p *Processor) Process(contentType string, content []byte, config map[string]ctypes.ConfigValue) (string, []byte, error) {

	p.logger.Printf("%s Processor started {%s}", pluginName, contentType)
	expression := config["expression"].(ctypes.ConfigValueStr).Value

	re, err := regexp.Compile(expression)
	if err != nil {
		return contentType, nil, fmt.Errorf(errRegexpCompilationError, expression, err)
	}

	tag := config["tag"].(ctypes.ConfigValueStr).Value

	var metrics []plugin.MetricType

	//Decodes the content into pluginMetricType
	dec := gob.NewDecoder(bytes.NewBuffer(content))
	if err := dec.Decode(&metrics); err != nil {
		p.logger.Printf("Error decoding: error=%v content=%v", err, content)
		return "", nil, err
	}

	metrics = filter(metrics, re, tag)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(metrics)
	return contentType, buf.Bytes(), nil
}

// Processor represents plugin for processing metrics retrieved from Mesos cluster
type Processor struct {
	logger *log.Logger
}

func filter(metrics []plugin.MetricType, re *regexp.Regexp, tag string) []plugin.MetricType {
	for i := range metrics {
		ns := metrics[i].Namespace()
		for j, nsElement := range ns {
			if re.MatchString(nsElement.Value) {
				metrics[i].Namespace_ = append(ns[:j], ns[j+1:]...)
				metrics[i].Tags()[tag] = nsElement.Value
			}
		}
	}

	return metrics
}
