'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-alert.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-alert.cjs.dev.js");
}
