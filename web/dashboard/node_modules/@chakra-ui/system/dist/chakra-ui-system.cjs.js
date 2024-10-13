'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-system.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-system.cjs.dev.js");
}
