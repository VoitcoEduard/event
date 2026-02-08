import React from 'react';
import { StyleSheet, Text, View, TouchableOpacity } from 'react-native';
import { StatusBar } from 'expo-status-bar';

export default function MainScreen({ onNavigate }) {

    return (
        <View style={styles.container}>
            <StatusBar style="dark" />
            <View style={styles.content}>
                <Text style={styles.title}>Hello from Expo</Text>
                <Text style={styles.subtitle}>Minimalist Design</Text>

                <View style={styles.card}>
                    <Text style={styles.cardText}>
                        Welcome to your new app experience. Clean, fast, and simple.
                    </Text>
                </View>

                <TouchableOpacity
                    style={styles.button}
                    onPress={() => onNavigate('admin')}
                    activeOpacity={0.8}
                >
                    <Text style={styles.buttonText}>Go to Admin Dashboard</Text>
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#FAFAFA', // Lightest gray, almost white
        alignItems: 'center',
        justifyContent: 'center',
    },
    content: {
        width: '90%',
        maxWidth: 400,
        alignItems: 'center',
    },
    title: {
        fontSize: 32,
        fontWeight: '800', // Extra bold
        color: '#1A1A1A', // Nearly black
        marginBottom: 8,
        letterSpacing: -0.5,
    },
    subtitle: {
        fontSize: 16,
        color: '#666666',
        marginBottom: 40,
        fontWeight: '500',
        letterSpacing: 0.5,
        textTransform: 'uppercase',
    },
    card: {
        backgroundColor: '#FFFFFF',
        padding: 24,
        borderRadius: 16,
        width: '100%',
        shadowColor: '#000',
        shadowOffset: {
            width: 0,
            height: 4,
        },
        shadowOpacity: 0.05,
        shadowRadius: 12,
        elevation: 3,
        marginBottom: 32,
        borderWidth: 1,
        borderColor: '#F0F0F0',
    },
    cardText: {
        fontSize: 16,
        color: '#444444',
        lineHeight: 24,
        textAlign: 'center',
    },
    button: {
        backgroundColor: '#1A1A1A',
        paddingVertical: 16,
        paddingHorizontal: 32,
        borderRadius: 12,
        width: '100%',
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: {
            width: 0,
            height: 4,
        },
        shadowOpacity: 0.1,
        shadowRadius: 8,
        elevation: 4,
    },
    buttonText: {
        color: '#FFFFFF',
        fontSize: 16,
        fontWeight: '600',
        letterSpacing: 0.5,
    },
});
