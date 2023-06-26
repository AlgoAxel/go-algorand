// Copyright (C) 2019-2023 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

//nolint:revive // ignore linter advice to add comments to exported functions
package logging

import (
	"io"

	"github.com/algorand/go-algorand/logging/telemetryspec"
	"github.com/sirupsen/logrus"
)

// NoopLogger does nothing
type NoopLogger struct{}

func (NoopLogger) Debug(...interface{})                      {}
func (NoopLogger) Debugln(...interface{})                    {}
func (NoopLogger) Debugf(string, ...interface{})             {}
func (NoopLogger) Info(...interface{})                       {}
func (NoopLogger) Infoln(...interface{})                     {}
func (NoopLogger) Infof(string, ...interface{})              {}
func (NoopLogger) Warn(...interface{})                       {}
func (NoopLogger) Warnln(...interface{})                     {}
func (NoopLogger) Warnf(string, ...interface{})              {}
func (NoopLogger) Error(...interface{})                      {}
func (NoopLogger) Errorln(...interface{})                    {}
func (NoopLogger) Errorf(string, ...interface{})             {}
func (NoopLogger) Fatal(...interface{})                      {}
func (NoopLogger) Fatalln(...interface{})                    {}
func (NoopLogger) Fatalf(string, ...interface{})             {}
func (NoopLogger) Panic(...interface{})                      {}
func (NoopLogger) Panicln(...interface{})                    {}
func (NoopLogger) Panicf(string, ...interface{})             {}
func (NoopLogger) With(key string, value interface{}) Logger { return NoopLogger{} }
func (NoopLogger) WithFields(Fields) Logger                  { return NoopLogger{} }
func (NoopLogger) SetLevel(Level)                            {}
func (NoopLogger) GetLevel() Level                           { return Level(Error) }
func (NoopLogger) SetOutput(io.Writer)                       {}
func (NoopLogger) SetJSONFormatter()                         {}
func (NoopLogger) IsLevelEnabled(level Level) bool           { return false }
func (NoopLogger) AddHook(hook logrus.Hook)                  {}
func (NoopLogger) EnableTelemetry(cfg TelemetryConfig) error { return nil }
func (NoopLogger) UpdateTelemetryURI(uri string) error       { return nil }
func (NoopLogger) GetTelemetryEnabled() bool                 { return false }
func (NoopLogger) GetTelemetryUploadingEnabled() bool        { return false }
func (NoopLogger) Metrics(category telemetryspec.Category, metrics telemetryspec.MetricDetails, details interface{}) {
}
func (NoopLogger) Event(category telemetryspec.Category, identifier telemetryspec.Event) {}
func (NoopLogger) EventWithDetails(category telemetryspec.Category, identifier telemetryspec.Event, details interface{}) {
}
func (NoopLogger) GetTelemetrySession() string { return "" }
func (NoopLogger) GetTelemetryGUID() string    { return "" }
func (NoopLogger) GetInstanceName() string     { return "" }
func (NoopLogger) GetTelemetryURI() string     { return "" }
func (NoopLogger) CloseTelemetry()             {}
