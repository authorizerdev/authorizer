'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-portal.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-portal.cjs.dev.js");
}
