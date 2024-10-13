'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-skeleton.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-skeleton.cjs.dev.js");
}
