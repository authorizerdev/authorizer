'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-tabs.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-tabs.cjs.dev.js");
}
