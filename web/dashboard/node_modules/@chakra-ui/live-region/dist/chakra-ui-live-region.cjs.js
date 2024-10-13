'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-live-region.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-live-region.cjs.dev.js");
}
