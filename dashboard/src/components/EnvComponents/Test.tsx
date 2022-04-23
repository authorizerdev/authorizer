import React from "react";

const Test = () => {
  return (
    <div>
      ######### INSTANCE INFORMATION #########
      {/* <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Your instance information
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Client ID</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={() => {}}
              inputType={TextInputType.CLIENT_ID}
              placeholder="Client ID"
              readOnly={true}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Client Secret</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.CLIENT_SECRET}
              placeholder="Client Secret"
              readOnly={true}
            />
          </Center>
        </Flex>
      </Stack> */}
      ######### SOCIAL MEDIA LOGIN #########
      {/* <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Social Media Logins
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Center
            w="50px"
            marginRight="1.5%"
            border="1px solid #e2e8f0"
            borderRadius="5px"
          >
            <FaGoogle style={{ color: "#8c8c8c" }} />
          </Center>
          <Center w="45%" marginRight="1.5%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={TextInputType.GOOGLE_CLIENT_ID}
              placeholder="Google Client ID"
            />
          </Center>
          <Center w="45%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.GOOGLE_CLIENT_SECRET}
              placeholder="Google Secret"
            />
          </Center>
        </Flex>
        <Flex>
          <Center
            w="50px"
            marginRight="1.5%"
            border="1px solid #e2e8f0"
            borderRadius="5px"
          >
            <FaGithub style={{ color: "#8c8c8c" }} />
          </Center>
          <Center w="45%" marginRight="1.5%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={TextInputType.GITHUB_CLIENT_ID}
              placeholder="Github Client ID"
            />
          </Center>
          <Center w="45%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.GITHUB_CLIENT_SECRET}
              placeholder="Github Secret"
            />
          </Center>
        </Flex>
        <Flex>
          <Center
            w="50px"
            marginRight="1.5%"
            border="1px solid #e2e8f0"
            borderRadius="5px"
          >
            <FaFacebookF style={{ color: "#8c8c8c" }} />
          </Center>
          <Center w="45%" marginRight="1.5%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={TextInputType.FACEBOOK_CLIENT_ID}
              placeholder="Facebook Client ID"
            />
          </Center>
          <Center w="45%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.FACEBOOK_CLIENT_SECRET}
              placeholder="Facebook Secret"
            />
          </Center>
        </Flex>
      </Stack> */}
      ######### ROLES #########
      {/* <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Roles
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Roles:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={ArrayInputType.ROLES}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Default Roles:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={ArrayInputType.DEFAULT_ROLES}
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Protected Roles:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={ArrayInputType.PROTECTED_ROLES}
            />
          </Center>
        </Flex>
      </Stack> */}
      ######### JWT COnFIGURATION #########
      {/* <Flex
        width="100%"
        justifyContent="space-between"
        alignItems="center"
        paddingTop="2%"
      >
        <Text fontSize="md" fontWeight="bold">
          JWT (JSON Web Tokens) Configurations
        </Text>
        <Flex>
          <GenerateKeysModal
            jwtType={envVariables.JWT_TYPE}
            getData={getData}
          />
        </Flex>
      </Flex>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">JWT Type:</Text>
          </Flex>
          <Flex w="70%">
            <InputField
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={SelectInputType.JWT_TYPE}
              value={SelectInputType.JWT_TYPE}
              options={{
                ...HMACEncryptionType,
                ...RSAEncryptionType,
                ...ECDSAEncryptionType,
              }}
            />
          </Flex>
        </Flex>
        {Object.values(HMACEncryptionType).includes(envVariables.JWT_TYPE) ? (
          <Flex>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">JWT Secret</Text>
            </Flex>
            <Center w="70%">
              <InputField
                variables={envVariables}
                setVariables={setEnvVariables}
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
                  variables={envVariables}
                  setVariables={setEnvVariables}
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
                  variables={envVariables}
                  setVariables={setEnvVariables}
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
              variables={envVariables}
              setVariables={setEnvVariables}
              inputType={TextInputType.JWT_ROLE_CLAIM}
            />
          </Center>
        </Flex>
      </Stack> */}
    </div>
  );
};

export default Test;
