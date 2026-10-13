// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build !darwin

package localfiles

import "context"

func (c *Capture) Save(context.Context, string) error { return ErrUnsupportedTarget }

func saveCapture(context.Context, *Capture, string, saveHooks) error { return ErrUnsupportedTarget }

type saveHooks struct{}
