/**
 * Created by igor on 29.08.16.
 */

"use strict";

const fsExtra = require('fs-extra'),
    fs = require('fs'),
    log = require(__appRoot + '/lib/log')(module),
    CodeError = require(__appRoot + '/lib/error'),
    path = require('path'),
    helper = require('./helper'),
    TYPE_ID = 0;

module.exports = class LocalStorage {

    constructor (conf, mask) {
        this.rootPath = conf.fileRoot;
        this.mask = mask;
        this.name = "local";
    }

    get (fileDb, options, cb) {
        fs.lstat(fileDb.path, function (err, stat) {
            if (err)
                return cb(err);

            if (!stat.isFile()) {
                return cb(new CodeError(404, 'Bad file.'))
            }

            let readable;

            if (options.range && options.range.Start < options.range.End) {
                readable = fs.createReadStream(fileDb.path, {flags: 'r', start: options.range.Start, end: options.range.End })
            } else {
                readable = fs.createReadStream(fileDb.path, {flags: 'r'});
            }

            if (options.skipOpen) {
                return cb(null, readable);
            }
            readable.on('open', () => {
                return cb(null, readable);
            })
        });
    }

    copyTo (fileDb, to, cb) {
        this.get(fileDb, {skipOpen: true}, (err, stream) => {
            if (err)
                return cb(err);

            to.save(fileDb, {stream}, cb);
        });
    }

    getFilePath(domain, fileName) {
        return path.join(this.rootPath, helper.getPath(this.mask, domain, fileName));
    }

    save (fileConf, option = {}, cb) {
        const pathFolder = path.join(this.rootPath, helper.getPath(this.mask, fileConf.domain));
        fsExtra.ensureDir(pathFolder, (err) => {
            if (err)
                return cb(err);

            if (option.stream) {
                let filePath = `${pathFolder}/${fileConf.uuid}_${fileConf.name}.${fileConf.applicationName}`;
                const wStream = fs.createWriteStream(filePath);
                option.stream.on("end", function(ex) {
                    log.trace(`Save stream file: ${filePath}`);
                    return cb(null, {
                        path: filePath,
                        type: TYPE_ID
                    })
                });

                option.stream.pipe(wStream)
            } else {
                let filePath = pathFolder + '/' + fileConf.name;
                copyFile(fileConf.path, filePath, () => {
                    if (err)
                        return cb(err);
                    log.trace(`Save file: ${filePath}`);
                    return cb(null, {
                        path: filePath,
                        type: TYPE_ID
                    })
                });
            }
        })
    }

    del (fileConf, cb) {
        fs.lstat(fileConf.path, function (err, stat) {
            if (err)
                return cb(err);

            if (!stat.isFile()) {
                return cb(new CodeError(404, 'Bad file.'))
            }
            log.debug(`Delete file ${fileConf.path}`);

            fs.unlink(fileConf.path, cb)
        });
    }

    existsFile (fileConf, cb) {
        fs.lstat(fileConf.path, (err, stats) => {
            if (err)
                return cb(null, false);

            return cb(null, stats.isFile())
        });
    }

    checkConfig (conf = {}, mask) {
        return this.mask == mask && this.rootPath == conf.fileRoot
    }
};

function copyFile(source, target, cb) {
    let cbCalled = false;

    let rd = fs.createReadStream(source);
    rd.on("error", function(err) {
        done(err);
    });
    var wr = fs.createWriteStream(target);
    wr.on("error", function(err) {
        done(err);
    });
    wr.on("close", function(ex) {
        done();
    });
    rd.pipe(wr);

    function done(err) {
        if (!cbCalled) {
            cb(err);
            cbCalled = true;
        }
    }
}