package domain

import "errors"

// SeatQuery selects every seat of one student, optionally narrowed by round and sheet.
type SeatQuery struct {
	StudentID string
	Round     *string
	Sheet     *string
}

// ErrRoundNotFound is returned for an operation on an unknown round.
var ErrRoundNotFound = errors.New("round not found")
