'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-stat.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-stat.cjs.dev.js");
}
