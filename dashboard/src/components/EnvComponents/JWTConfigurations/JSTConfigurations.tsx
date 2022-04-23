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
import {
  HiddenInputType,
  TextInputType,
  TextAreaInputType,
} from "../../../constants";
import GenerateKeysModal from "../../GenerateKeysModal";
import InputField from "../../InputField";

const JSTConfigurations = ({
  variables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
  SelectInputType,
  getData,
  HMACEncryptionType,
  RSAEncryptionType,
  ECDSAEncryptionType,
}: any) => {
  return (
    <div>
      {" "}
      <Flex
        width="100%"
        justifyContent="space-between"
        alignItems="center"
        paddingTop="2%"
      >
        <Text fontSize="md" fontWeight="bold">
          JWT (JSON Web Tokens) Configurations
        </Text>
        <Flex>
          <GenerateKeysModal jwtType={variables.JWT_TYPE} getData={getData} />
        </Flex>
      </Flex>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">JWT Type:</Text>
          </Flex>
          <Flex w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SelectInputType}
              value={SelectInputType}
              options={{
                ...HMACEncryptionType,
                ...RSAEncryptionType,
                ...ECDSAEncryptionType,
              }}
            />
          </Flex>
        </Flex>
        {Object.values(HMACEncryptionType).includes(variables.JWT_TYPE) ? (
          <Flex>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">JWT Secret</Text>
            </Flex>
            <Center w="70%">
              <InputField
                variables={variables}
                setVariables={setVariables}
                fieldVisibility={fieldVisibility}
                setFieldVisibility={setFieldVisibility}
                inputType={HiddenInputType.JWT_SECRET}
              />
            </Center>
          </Flex>
        ) : (
          <>
            <Flex>
              <Flex w="30%" justifyContent="start" alignItems="center">
                <Text fontSize="sm">Public Key</Text>
              </Flex>
              <Center w="70%">
                <InputField
                  variables={variables}
                  setVariables={setVariables}
                  inputType={TextAreaInputType.JWT_PUBLIC_KEY}
                  placeholder="Add public key here"
                  minH="25vh"
                />
              </Center>
            </Flex>
            <Flex>
              <Flex w="30%" justifyContent="start" alignItems="center">
                <Text fontSize="sm">Private Key</Text>
              </Flex>
              <Center w="70%">
                <InputField
                  variables={variables}
                  setVariables={setVariables}
                  inputType={TextAreaInputType.JWT_PRIVATE_KEY}
                  placeholder="Add private key here"
                  minH="25vh"
                />
              </Center>
            </Flex>
          </>
        )}
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">JWT Role Claim:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.JWT_ROLE_CLAIM}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default JSTConfigurations;
