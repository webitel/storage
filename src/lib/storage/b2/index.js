/**
 * Created by igor on 29.08.16.
 */

"use strict";

const log = require(__appRoot + '/lib/log')(module),
    CodeError = require(__appRoot + '/lib/error'),
    helper = require('../helper'),
    B2 = require('./b2'),
    async = require('async'),
    TYPE_ID = 2;

module.exports = class B2Storage {

    constructor (conf, mask) {
        this.accountId = conf.accountId;
        this.bucketId = conf.bucketId;
        this.bucketName = conf.bucketName;
        this.applicationKey = conf.applicationKey;
        this.mask = mask || "$Y/$M/$D/$H";
        this._expireToken = 0;
        this.name = "b2";
        this._uploadQueue = async.queue( (data, cb) => {
            const {fileConf, options} = data;
            B2.saveFile(this._authParams, fileConf, options, helper.getPath(this.mask, fileConf.domain, fileConf.name), cb);
        });
        this._uploadQueue.drain = () => {
            log.debug('All upload files done');
        };
        this._auth();
    }

    _isAuth (cb) {
        if (this._expireToken <= Date.now()) {
            return this._auth(cb);
        } else {
            return cb(null)
        }
    }

    _auth (cb) {
        let conf = {
            accountId: this.accountId,
            applicationKey: this.applicationKey,
            bucketId: this.bucketId,
            bucketName: this.bucketName
        };

        B2.auth(conf, (err, authParam) => {
            // token validate 24h
            // storage re-auth 6h
            // 1000 * 60 * 60 * 6 = 21600000
            if (err) {
                log.error(err);
                return cb && cb(err);
            }

            this._expireToken = Date.now() + 21600000;
            log.trace(`Confirm auth ${conf.accountId}`);
            this._authParams = authParam;
            return cb && cb();
        });
    }

    get (fileDb, options, cb) {
        this._isAuth((err) => {
            if (err)
                return cb(err);

            return B2.getFile(this._authParams, fileDb, options.range, cb);
        })
    }

    copyTo (fileDb, to, cb) {
        this.get(fileDb, {}, (err, stream) => {
            if (err)
                return cb(err);

            to.save(fileDb, {stream}, cb);
        });
    }

    save (fileConf, options, cb) {
        this._isAuth((err) => {
            if (err)
                return cb(err);

            return this._uploadQueue.push({fileConf, options}, cb)
        });
    }

    del (fileConf, cb) {
        this._isAuth((err) => {
            if (err)
                return cb(err);

            return B2.delFile(this._authParams, fileConf, cb);
        });
    }

    existsFile (fileConf, cb) {
        if (!fileConf.storageFileId)  {
            return cb(null, false);
        }

        this._isAuth((err) => {
            if (err)
                return cb(err);

            B2.getFileInfo(this._authParams, fileConf.storageFileId, (err) => {
                if (err && err.status === 404) {
                    return cb(null, false)
                } else if (err) {
                    return cb(err);
                } else {
                    return cb(null, true);
                }
            })
        });
    }

    checkConfig (conf = {}, mask) {
        return this.mask == mask && this.accountId == conf.accountId && this.bucketId == conf.bucketId
            && this.bucketName == conf.bucketName && this.applicationKey == conf.applicationKey
    }
};