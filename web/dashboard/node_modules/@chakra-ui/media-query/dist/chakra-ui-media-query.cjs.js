'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-media-query.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-media-query.cjs.dev.js");
}
