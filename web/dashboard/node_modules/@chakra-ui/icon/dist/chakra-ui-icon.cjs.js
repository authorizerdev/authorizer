'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-icon.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-icon.cjs.dev.js");
}
