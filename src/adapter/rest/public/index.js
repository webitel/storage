/**
 * Created by igor on 23.08.16.
 */

"use strict";

module.exports = addRoutes;

/**
 * Adds routes to the api.
 */
function addRoutes(api) {
    require('./acl').addRoutes(api);
    require('./recordFile').addRoutes(api);
    require('./cdr').addRoutes(api);
    require('./media').addRoutes(api);
    require('./tcpDump').addRoutes(api);
    require('./statistic').addRoutes(api);
}