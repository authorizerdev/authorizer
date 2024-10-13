'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-tooltip.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-tooltip.cjs.dev.js");
}
