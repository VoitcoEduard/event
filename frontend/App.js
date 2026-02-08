import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import MainScreen from './MainScreen';
import AdminScreen from './AdminScreen';

export default function App() {
  const [currentScreen, setCurrentScreen] = useState('main');

  const renderScreen = () => {
    switch (currentScreen) {
      case 'admin':
        return <AdminScreen onNavigate={setCurrentScreen} />;
      case 'main':
      default:
        return <MainScreen onNavigate={setCurrentScreen} />;
    }
  };

  return (
    <View style={styles.container}>
      {renderScreen()}
      <StatusBar style="auto" />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
});
