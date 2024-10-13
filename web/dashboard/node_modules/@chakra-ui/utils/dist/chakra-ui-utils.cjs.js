'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-utils.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-utils.cjs.dev.js");
}
