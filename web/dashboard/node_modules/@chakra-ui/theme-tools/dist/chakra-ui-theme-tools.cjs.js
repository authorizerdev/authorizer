'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-theme-tools.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-theme-tools.cjs.dev.js");
}
