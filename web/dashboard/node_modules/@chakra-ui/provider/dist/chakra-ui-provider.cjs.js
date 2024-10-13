'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-provider.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-provider.cjs.dev.js");
}
