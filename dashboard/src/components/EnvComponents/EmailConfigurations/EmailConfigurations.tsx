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
import { TextInputType, HiddenInputType } from "../../../constants";
const EmailConfigurations = ({
  variables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
}: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Email Configurations
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">SMTP Host:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.SMTP_HOST}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">SMTP Port:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.SMTP_PORT}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">SMTP Username:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.SMTP_USERNAME}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">SMTP Password:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.SMTP_PASSWORD}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">From Email:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.SENDER_EMAIL}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default EmailConfigurations;
