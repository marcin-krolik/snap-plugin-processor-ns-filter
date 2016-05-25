// +build unit

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
	"regexp"
	"testing"

	"github.com/intelsdi-x/snap/control/plugin"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
)

func TestMeta(t *testing.T) {
	Convey("Meta should return plugin meta data", t, func() {
		meta := Meta()
		So(meta.Name, ShouldEqual, pluginName)
		So(meta.Version, ShouldEqual, pluginVersion)
		So(meta.Type, ShouldResemble, pluginType)
	})
}

func TestNew(t *testing.T) {
	proc := New()
	Convey("Create ns-filter processor", t, func() {
		Convey("So proc should not be nil", func() {
			So(proc, ShouldNotBeNil)
		})
		Convey("So proc should be of type processor", func() {
			So(proc, ShouldHaveSameTypeAs, &Processor{})
		})
	})
}

func TestPolicyProperConfig(t *testing.T) {
	Convey("proc.GetConfigPolicy should return a config policy", t, func() {
		proc := New()
		configPolicy, _ := proc.GetConfigPolicy()
		Convey("So config policy should be a cpolicy.ConfigPolicy", func() {
			So(configPolicy, ShouldHaveSameTypeAs, &cpolicy.ConfigPolicy{})
		})

		Convey("Proper config provided", func() {
			testConfig := make(map[string]ctypes.ConfigValue)
			testConfig["expression"] = ctypes.ConfigValueStr{Value: "foo"}
			testConfig["tag"] = ctypes.ConfigValueStr{Value: "bar"}
			cfg, errs := configPolicy.Get([]string{"/"}).Process(testConfig)

			Convey("So config policy should process testConfig and return a config", func() {
				So(cfg, ShouldNotBeNil)
			})
			Convey("So testConfig processing should return no errors", func() {
				So(errs.HasErrors(), ShouldBeFalse)
			})
		})
	})
}

func TestPolicyBadConfig(t *testing.T) {
	Convey("proc.GetConfigPolicy should return a config policy", t, func() {
		proc := New()
		configPolicy, _ := proc.GetConfigPolicy()
		Convey("So config policy should be a cpolicy.ConfigPolicy", func() {
			So(configPolicy, ShouldHaveSameTypeAs, &cpolicy.ConfigPolicy{})
		})

		Convey("Expression is not provided", func() {
			testConfigBad := make(map[string]ctypes.ConfigValue)
			testConfigBad["exp"] = ctypes.ConfigValueStr{Value: "foo"}
			testConfigBad["tag"] = ctypes.ConfigValueStr{Value: "bar"}
			cfg, errs := configPolicy.Get([]string{"/"}).Process(testConfigBad)

			Convey("So config policy should process testConfigBad and return a config", func() {
				So(cfg, ShouldBeNil)
			})
			Convey("So testConfig processing should return no errors", func() {
				So(errs.HasErrors(), ShouldBeTrue)
			})
		})

		Convey("Tag is not provided", func() {
			testConfigBad := make(map[string]ctypes.ConfigValue)
			testConfigBad["expression"] = ctypes.ConfigValueStr{Value: "foo"}
			cfg, errs := configPolicy.Get([]string{"/"}).Process(testConfigBad)

			Convey("So config policy should process testConfigBad and return a config", func() {
				So(cfg, ShouldBeNil)
			})
			Convey("So testConfig processing should return no errors", func() {
				So(errs.HasErrors(), ShouldBeTrue)
			})
		})

		Convey("Expression and Tag are not provided", func() {
			testConfigBad := map[string]ctypes.ConfigValue{}
			cfg, errs := configPolicy.Get([]string{"/"}).Process(testConfigBad)

			Convey("So config policy should process testConfigBad and return a config", func() {
				So(cfg, ShouldBeNil)
			})
			Convey("So testConfig processing should return no errors", func() {
				So(errs.HasErrors(), ShouldBeTrue)
			})
		})
	})
}

func TestProcess(t *testing.T) {
	Convey("Filtering metrics with host IP in namespace", t, func() {
		metrics := []plugin.MetricType{
			plugin.MetricType{
				Namespace_: core.NewNamespace("intel", "foo", "1.1.1.2", "bar"),
				Tags_:      map[string]string{},
			},
			plugin.MetricType{
				Namespace_: core.NewNamespace("intel", "foo", "100.1.1.100", "bar"),
				Tags_:      map[string]string{"faz": "qaz"},
			},
		}

		expected := []plugin.MetricType{
			plugin.MetricType{
				Namespace_: core.NewNamespace("intel", "foo", "bar"),
				Tags_:      map[string]string{"ip": "1.1.1.2"},
			},
			plugin.MetricType{
				Namespace_: core.NewNamespace("intel", "foo", "bar"),
				Tags_:      map[string]string{"faz": "qaz", "ip": "100.1.1.100"},
			},
		}

		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)

		config := make(map[string]ctypes.ConfigValue)
		config["expression"] = ctypes.ConfigValueStr{Value: `([0-9]{1,3}\.){3}([0-9]{1,3})`}
		config["tag"] = ctypes.ConfigValueStr{Value: "ip"}

		processor := New()
		contentType, content, err := processor.Process("gob", buf.Bytes(), config)

		var decoded []plugin.MetricType
		dec := gob.NewDecoder(bytes.NewBuffer(content))
		dec.Decode(&decoded)

		So(contentType, ShouldEqual, "gob")
		So(decoded, ShouldResemble, expected)
		So(err, ShouldBeNil)

	})
}

func TestFilter(t *testing.T) {
	re, _ := regexp.Compile(`([0-9]{1,3}\.){3}([0-9]{1,3})`)
	Convey("Filter metrics from unwanted parts", t, func() {
		Convey("Nothing to remove", func() {
			metrics := []plugin.MetricType{
				plugin.MetricType{
					Namespace_: core.NewNamespace("intel", "foo", "bar"),
					Tags_:      map[string]string{},
				},
			}
			filter(metrics, re, "ip")
			So(metrics[0].Namespace().String(), ShouldEqual, core.NewNamespace("intel", "foo", "bar").String())
			So(metrics[0].Tags(), ShouldResemble, map[string]string{})
		})

		Convey("Remove IP 1.1.1.2, add as tag", func() {
			metrics := []plugin.MetricType{
				plugin.MetricType{
					Namespace_: core.NewNamespace("intel", "foo", "1.1.1.2", "bar"),
					Tags_:      map[string]string{},
				},
			}

			filter(metrics, re, "ip")
			So(metrics[0].Namespace().String(), ShouldEqual, core.NewNamespace("intel", "foo", "bar").String())
			So(metrics[0].Tags(), ShouldResemble, map[string]string{"ip": "1.1.1.2"})
		})

		Convey("Remove IP 10.255.255.100, add as tag", func() {
			metrics := []plugin.MetricType{
				plugin.MetricType{
					Namespace_: core.NewNamespace("intel", "foo", "10.255.255.100", "bar"),
					Tags_:      map[string]string{},
				},
			}

			filter(metrics, re, "ip")
			So(metrics[0].Namespace().String(), ShouldEqual, core.NewNamespace("intel", "foo", "bar").String())
			So(metrics[0].Tags(), ShouldResemble, map[string]string{"ip": "10.255.255.100"})
		})

		Convey("Remove IP 100.1.1.100, add as tag, preserve tags", func() {
			metrics := []plugin.MetricType{
				plugin.MetricType{
					Namespace_: core.NewNamespace("intel", "foo", "100.1.1.100", "bar"),
					Tags_:      map[string]string{"faz": "qaz"},
				},
			}

			filter(metrics, re, "ip")
			So(metrics[0].Namespace().String(), ShouldEqual, core.NewNamespace("intel", "foo", "bar").String())
			So(metrics[0].Tags(), ShouldResemble, map[string]string{"faz": "qaz", "ip": "100.1.1.100"})
		})

	})
}
