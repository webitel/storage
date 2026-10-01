package model

import "time"

var APP_SERVICE_NAME = "storage"

const APP_SERVICE_TTL = time.Second * 30
const APP_DEREGISTER_CRITICAL_TTL = time.Second * 60
