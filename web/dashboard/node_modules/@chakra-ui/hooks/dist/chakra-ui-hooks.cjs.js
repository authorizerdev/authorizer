'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-hooks.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-hooks.cjs.dev.js");
}
