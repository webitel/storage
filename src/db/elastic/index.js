/**
 * Created by igor on 30.08.16.
 */

"use strict";

const elasticsearch = require('elasticsearch'),
    EventEmitter2 = require('eventemitter2').EventEmitter2,
    log = require(__appRoot + '/lib/log')(module),
    setCustomAttribute = require(__appRoot + '/utils/cdr').setCustomAttribute,
    async = require('async')
;

const CDR_NAME = 'cdr*',
    MAX_RESULT_WINDOW = 2147483647;

class ElasticClient extends EventEmitter2 {
    constructor (config) {
        super();
        this.connected = false;
        this.config = config;

        this.client = new elasticsearch.Client({
            host: config.host,
            keepAlive: config.keepAlive || true,
            requestTimeout: config.requestTimeout || 500000
        });

        this.ping();
        this.pingId = null;
    }

    ping () {
        if (this.pingId)
            clearTimeout(this.pingId);

        this.client.ping({}, (err) => {
            if (err) {
                log.error(err);
                this.pingId = setTimeout(this.ping.bind(this), 1000);
                return null;
            }

            log.info(`Connect to elastic - OK`);
            this.connected = true;
            this.initTemplate();
            this.initMaxResultWindow();
            this.emit('elastic:connect', this);
        })
    }

    initTemplate () {
        const client = this.client;

        client.indices.getTemplate((err, res) => {
            if (err) {
                return log.error(err);
            }

            let elasticTemplatesNames = Object.keys(res),
                templates = this.config.templates || [],
                tasks = [],
                delTemplate = [];

            templates.forEach(function (template) {
                if (elasticTemplatesNames.indexOf(template.name) > -1) {
                    delTemplate.push(function (done) {
                        client.indices.deleteTemplate(
                            template,
                            function (err) {
                                if (err) {
                                    log.error(err);
                                } else {
                                    log.debug('Template %s deleted.', template.name)
                                }
                                done();
                            }
                        );
                    });
                }

                tasks.push(function (done) {
                    client.indices.putTemplate(
                        template,
                        function (err) {
                            if (err) {
                                log.error(err);
                            } else {
                                log.debug('Template %s - created.', template.name);
                            }
                            done();
                        }
                    );
                });
            });

            if (tasks.length > 0) {
                async.waterfall([].concat(delTemplate, tasks) , (err) => {
                    if (err)
                        return log.error(err);
                    return log.info(`Replace elastic template - OK`);
                });
            }
        });
    }

    initMaxResultWindow () {
        this.client.indices.getSettings({
            index: CDR_NAME,
            name: "index.max_result_window"
        }, (err, res) => {
            if (err) {
                return log.error(err);
            }

            let indexName = Object.keys(res);
            if (indexName.length > 0) {
                let indexSettings = res[indexName[0]] && res[indexName[0]].settings;
                let max_result_window = +(indexSettings && indexSettings.index && indexSettings.index.max_result_window);

                if (!max_result_window || max_result_window < 1000000) {
                    this.setIndexSettings()
                } else {
                    log.trace('Skip set max_result_window')
                }
            } else {
                this.setIndexSettings();
            }
        });
    }

    setIndexSettings () {
        this.client.indices.putSettings({
            index: CDR_NAME,
            body: {
                max_result_window: MAX_RESULT_WINDOW
            }
        }, (err, res) => {
            if (err)
                return log.error(err);

            log.info(`Set default max_result_window - success`);
        });
    }

    search (data, cb) {
        return this.client.search(data, cb);
    }

    insertCdr (doc, cb) {
        let currentDate = new Date(),
            indexName = `cdr-${currentDate.getMonth() + 1}.${currentDate.getFullYear()}`,
            _record = setCustomAttribute(doc),
            _id = _record._id.toString();
        delete _record._id;

        this.client.create({
            index: (indexName + (doc.variables.domain_name ? '-' + doc.variables.domain_name : '')).toLowerCase(),
            type: 'collection',
            id: _id,
            body: _record
        }, cb);
    }

    findByUuid (uuid, domain, cb) {
        this.client.search(
            {
                index: `cdr-*${domain ? '-' + domain : '' }`,
                size: 1,
                _source: false,
                body: {
                    "query": {
                        "term": {
                            "variables.uuid": uuid
                        }
                    }
                }
            },
            cb
        )
    }

    removeCdr (id, indexName = "", cb) {
        this.client.delete({
            index: indexName,
            type: 'collection',
            id: id
        }, cb)
    }
}
    
module.exports = ElasticClient;