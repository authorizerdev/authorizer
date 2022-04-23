import React from "react";
import {
  Box,
  Divider,
  Flex,
  Stack,
  Center,
  Text,
  Button,
  Input,
  InputGroup,
  InputRightElement,
  useToast,
} from "@chakra-ui/react";
import InputField from "../../InputField";
import { TextInputType, TextAreaInputType } from "../../../constants";

const AccessToken = ({ variables, setVariables }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Access Token
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Access Token Expiry Time:</Text>
          </Flex>
          <Flex w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.ACCESS_TOKEN_EXPIRY_TIME}
              placeholder="0h15m0s"
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" direction="column">
            <Text fontSize="sm">Custom Scripts:</Text>
            <Text fontSize="sm">Used to add custom fields in ID token</Text>
          </Flex>
          <Flex w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextAreaInputType.CUSTOM_ACCESS_TOKEN_SCRIPT}
              placeholder="Add script here"
              minH="25vh"
            />
          </Flex>
        </Flex>
      </Stack>
    </div>
  );
};

export default AccessToken;
