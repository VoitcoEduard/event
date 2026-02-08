import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, Animated, Clipboard, Alert, Platform, TextInput } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import QRCode from 'react-qr-code';

const API_URL = 'http://localhost:8080';

export default function AdminEventDetailsScreen({ event, onNavigate, onBack }) {
    const [currentEvent, setCurrentEvent] = useState(event);
    const [activeStatusAnim] = useState(new Animated.Value(event?.isActive ? 1 : 0));

    // Module Management State
    const [moduleModalVisible, setModuleModalVisible] = useState(false);
    const [newModuleName, setNewModuleName] = useState('');
    const [newModuleQuestion, setNewModuleQuestion] = useState('');

    useEffect(() => {
        // Refresh event data periodically or on mount to ensure status is up to date
        // If event is passed, use its ID. If not, we might need ID from elsewhere (e.g. params if we used routing)
        // For now, assume if event is null, we can't do much unless we have an ID. 
        if (event?.id) {
            fetchEventDetails();
        }
    }, [event]);

    useEffect(() => {
        if (currentEvent) {
            Animated.timing(activeStatusAnim, {
                toValue: currentEvent.isActive ? 1 : 0,
                duration: 300,
                useNativeDriver: false,
            }).start();
        }
    }, [currentEvent?.isActive]);

    if (!currentEvent) {
        return (
            <View style={styles.container}>
                <StatusBar style="light" />
                <View style={styles.header}>
                    <TouchableOpacity onPress={onBack} style={styles.backLink}>
                        <Text style={styles.backLinkText}>← Back</Text>
                    </TouchableOpacity>
                    <Text style={styles.headerTitle}>Event Dashboard</Text>
                </View>
                <View style={[styles.content, { justifyContent: 'center', alignItems: 'center' }]}>
                    <Text style={{ color: '#FFF' }}>Loading event details...</Text>
                </View>
            </View>
        );
    }

    const fetchEventDetails = async () => {
        try {
            // Fetch specific event details
            const response = await fetch(`${API_URL}/events/${event.id}`);
            if (response.ok) {
                const data = await response.json();
                setCurrentEvent(data);
            } else {
                // Fallback if specific endpoint doesn't exist
                const listResponse = await fetch(`${API_URL}/events`);
                if (listResponse.ok) {
                    const listData = await listResponse.json();
                    const found = listData.find(e => e.id === event.id);
                    if (found) setCurrentEvent(found);
                }
            }
        } catch (error) {
            console.error('Error fetching event details:', error);
        }
    };

    const handleToggleStatus = async () => {
        try {
            const response = await fetch(`${API_URL}/events/${currentEvent.id}/status`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ isActive: !currentEvent.isActive }),
            });

            if (response.ok) {
                const updatedEvent = await response.json();
                // If backend returns the updated event, use it. Otherwise toggle locally.
                // Assuming backend might just return 200 OK.
                setCurrentEvent(prev => ({ ...prev, isActive: !prev.isActive }));
                Alert.alert('Success', `Event ${!currentEvent.isActive ? 'started' : 'stopped'}`);
            } else {
                Alert.alert('Error', 'Failed to update status');
            }
        } catch (error) {
            console.error('Error updating status:', error);
            Alert.alert('Error', 'Failed to update status');
        }
    };

    const getShareLink = () => {
        if (typeof window !== 'undefined') {
            return `${window.location.origin}/?pin=${currentEvent.id}`;
        }
        return `http://yourapp.com/?pin=${currentEvent.id}`;
    };

    const copyToClipboard = () => {
        const shareLink = getShareLink();
        Clipboard.setString(shareLink);
        if (Platform.OS === 'web') {
            navigator.clipboard.writeText(shareLink).then(() => {
                Alert.alert('Copied', 'Event link copied to clipboard!');
            });
        } else {
            Alert.alert('Copied', 'Event link copied to clipboard!');
        }
    };

    const handleAddModule = async () => {
        if (!newModuleName.trim() || !newModuleQuestion.trim()) {
            Alert.alert('Error', 'Please fill in all fields');
            return;
        }

        try {
            const moduleData = {
                type: 'form',
                name: newModuleName,
                questions: [
                    {
                        id: 'q1', // Simple ID for now
                        text: newModuleQuestion,
                        type: 'text'
                    }
                ]
            };

            const response = await fetch(`${API_URL}/events/${currentEvent.id}/modules`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(moduleData),
            });

            if (response.ok) {
                const newModule = await response.json();
                // Optimistically update or re-fetch
                fetchEventDetails();
                setModuleModalVisible(false);
                setNewModuleName('');
                setNewModuleQuestion('');
                Alert.alert('Success', 'Module added successfully');
            } else {
                const errText = await response.text();
                Alert.alert('Error', 'Failed to add module: ' + errText);
            }
        } catch (error) {
            console.error('Error adding module:', error);
            Alert.alert('Error', 'Failed to add module');
        }
    };

    const handleUpdateModuleStatus = async (moduleId, newStatus) => {
        try {
            const response = await fetch(`${API_URL}/events/${currentEvent.id}/modules/${moduleId}/status`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ status: newStatus }),
            });

            if (response.ok) {
                fetchEventDetails(); // Refresh to get updated status
            } else {
                Alert.alert('Error', 'Failed to update module status');
            }
        } catch (error) {
            console.error('Error updating module status:', error);
            Alert.alert('Error', 'Failed to update module status');
        }
    };

    const StatusIndicator = () => {
        const backgroundColor = activeStatusAnim.interpolate({
            inputRange: [0, 1],
            outputRange: ['rgba(255, 69, 58, 0.2)', 'rgba(48, 209, 88, 0.2)']
        });
        const textColor = activeStatusAnim.interpolate({
            inputRange: [0, 1],
            outputRange: ['#FF453A', '#30D158']
        });

        return (
            <Animated.View style={[styles.statusBadge, { backgroundColor }]}>
                <Animated.Text style={[styles.statusText, { color: textColor }]}>
                    {currentEvent.isActive ? 'ACTIVE - RUNNING' : 'INACTIVE - STOPPED'}
                </Animated.Text>
            </Animated.View>
        );
    };

    return (
        <View style={styles.container}>
            <StatusBar style="light" />

            <View style={styles.header}>
                <TouchableOpacity onPress={onBack} style={styles.backLink}>
                    <Text style={styles.backLinkText}>← Back to Dashboard</Text>
                </TouchableOpacity>
                <Text style={styles.headerTitle}>Event Dashboard</Text>
                <Text style={styles.headerSubtitle}>{currentEvent.name}</Text>
            </View>

            <ScrollView style={styles.content} contentContainerStyle={styles.scrollContent}>

                {/* 1. Event Info Card */}
                <View style={styles.card}>
                    <View style={styles.cardHeader}>
                        <Text style={styles.cardTitle}>Event Details</Text>
                        <StatusIndicator />
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Event Name:</Text>
                        <Text style={styles.value}>{currentEvent.name}</Text>
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Description:</Text>
                        <Text style={styles.value}>{currentEvent.description}</Text>
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>PIN Code:</Text>
                        <View style={styles.pinContainer}>
                            <Text style={styles.pinText}>{currentEvent.id}</Text>
                        </View>
                    </View>
                </View>

                {/* 2. Share Access Card (New) */}
                <View style={styles.card}>
                    <Text style={styles.cardTitle}>Share Access</Text>
                    <Text style={styles.cardDescription}>
                        Share this QR code or link with participants to let them join the event directly.
                    </Text>

                    <View style={styles.qrCodeContainer}>
                        {Platform.OS === 'web' ? (
                            <div style={{ background: 'white', padding: '16px', borderRadius: '16px' }}>
                                <QRCode
                                    value={getShareLink()}
                                    size={200}
                                    style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                                    viewBox={`0 0 256 256`}
                                />
                            </div>
                        ) : (
                            <View style={styles.qrPlaceholder}>
                                <Text style={styles.qrPlaceholderText}>QR Code available on Web</Text>
                            </View>
                        )}
                    </View>

                    <TouchableOpacity style={styles.copyButton} onPress={copyToClipboard}>
                        <Text style={styles.copyButtonText}>Copy Shareable Link</Text>
                    </TouchableOpacity>
                </View>

                {/* 3. Modules Management Card */}
                <View style={[styles.card, { minWidth: 600 }]}>
                    <View style={styles.cardHeader}>
                        <Text style={styles.cardTitle}>Event Modules</Text>
                        <TouchableOpacity
                            style={styles.addButton}
                            onPress={() => setModuleModalVisible(true)}
                        >
                            <Text style={styles.addButtonText}>+ Add Module</Text>
                        </TouchableOpacity>
                    </View>
                    <Text style={styles.cardDescription}>
                        Manage interactive modules for this event. Only one module can be active at a time to focus user attention.
                    </Text>

                    {currentEvent.modules && currentEvent.modules.length > 0 ? (
                        <View style={styles.modulesList}>
                            {currentEvent.modules.map((module, index) => (
                                <View key={index} style={styles.moduleItem}>
                                    <View style={styles.moduleInfo}>
                                        <Text style={styles.moduleName}>{module.name}</Text>
                                        <Text style={styles.moduleType}>{module.type.toUpperCase()}</Text>
                                        <View style={[styles.moduleStatusBadge,
                                        module.status === 'active' ? { backgroundColor: 'rgba(48, 209, 88, 0.2)' } :
                                            module.status === 'ended' ? { backgroundColor: 'rgba(255, 69, 58, 0.1)' } :
                                                { backgroundColor: 'rgba(255, 255, 255, 0.1)' }
                                        ]}>
                                            <Text style={[styles.moduleStatusText,
                                            module.status === 'active' ? { color: '#30D158' } :
                                                module.status === 'ended' ? { color: '#FF453A' } :
                                                    { color: '#AAA' }
                                            ]}>
                                                {module.status.toUpperCase()}
                                            </Text>
                                        </View>
                                    </View>
                                    <View style={styles.moduleActions}>
                                        {module.status === 'pending' && (
                                            <TouchableOpacity
                                                style={[styles.moduleBtn, styles.activateBtn]}
                                                onPress={() => handleUpdateModuleStatus(module.id, 'active')}
                                            >
                                                <Text style={styles.moduleBtnText}>Activate</Text>
                                            </TouchableOpacity>
                                        )}
                                        {module.status === 'active' && (
                                            <TouchableOpacity
                                                style={[styles.moduleBtn, styles.endBtn]}
                                                onPress={() => handleUpdateModuleStatus(module.id, 'ended')}
                                            >
                                                <Text style={styles.moduleBtnText}>End</Text>
                                            </TouchableOpacity>
                                        )}
                                        {module.status === 'ended' && (
                                            <Text style={styles.endedText}>Completed</Text>
                                        )}
                                    </View>
                                </View>
                            ))}
                        </View>
                    ) : (
                        <View style={styles.emptyModules}>
                            <Text style={styles.emptyModulesText}>No modules added yet.</Text>
                        </View>
                    )}

                    {/* Add Module Form (Inline for now) */}
                    {moduleModalVisible && (
                        <View style={styles.addModuleForm}>
                            <Text style={styles.formTitle}>Add New Module</Text>
                            <Text style={styles.label}>Module Type</Text>
                            <View style={styles.typeSelector}>
                                <TouchableOpacity style={[styles.typeOption, styles.selectedType]}>
                                    <Text style={styles.typeOptionText}>Form / Survey</Text>
                                </TouchableOpacity>
                            </View>

                            <Text style={styles.label}>Module Name</Text>
                            <TextInput
                                style={styles.input}
                                placeholder="e.g. Feedback Form"
                                placeholderTextColor="#666"
                                value={newModuleName}
                                onChangeText={setNewModuleName}
                            />

                            <Text style={styles.label}>Question</Text>
                            <TextInput
                                style={styles.input}
                                placeholder="e.g. What did you like most?"
                                placeholderTextColor="#666"
                                value={newModuleQuestion}
                                onChangeText={setNewModuleQuestion}
                            />

                            <View style={styles.formActions}>
                                <TouchableOpacity
                                    style={styles.cancelButton}
                                    onPress={() => setModuleModalVisible(false)}
                                >
                                    <Text style={styles.cancelButtonText}>Cancel</Text>
                                </TouchableOpacity>
                                <TouchableOpacity
                                    style={styles.saveButton}
                                    onPress={handleAddModule}
                                >
                                    <Text style={styles.saveButtonText}>Add Module</Text>
                                </TouchableOpacity>
                            </View>
                        </View>
                    )}
                </View>

                {/* 3. Control Card */}
                <View style={styles.card}>
                    <Text style={styles.cardTitle}>Event Controls</Text>
                    <Text style={styles.cardDescription}>
                        Manage the lifecycle of the event. Stopping the event will disconnect all users.
                    </Text>

                    <TouchableOpacity
                        style={[styles.actionButton, currentEvent.isActive ? styles.stopButton : styles.startButton]}
                        onPress={handleToggleStatus}
                    >
                        <Text style={styles.actionButtonText}>
                            {currentEvent.isActive ? 'STOP EVENT' : 'START EVENT'}
                        </Text>
                    </TouchableOpacity>
                </View>

                {/* Placeholder for more cards */}
                <View style={[styles.card, styles.statsCard]}>
                    <Text style={styles.cardTitle}>Quick Stats</Text>
                    <View style={styles.statsGrid}>
                        <View style={styles.statItem}>
                            <Text style={styles.statNumber}>0</Text>
                            <Text style={styles.statLabel}>Active Users</Text>
                        </View>
                        <View style={styles.statItem}>
                            <Text style={styles.statNumber}>00:00</Text>
                            <Text style={styles.statLabel}>Duration</Text>
                        </View>
                    </View>
                </View>

            </ScrollView>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
    },
    header: {
        paddingTop: 60,
        paddingBottom: 20,
        paddingHorizontal: 24,
        backgroundColor: '#1E1E1E',
        borderBottomWidth: 1,
        borderBottomColor: '#2C2C2C',
    },
    backLink: {
        marginBottom: 10,
    },
    backLinkText: {
        color: '#007AFF',
        fontSize: 16,
    },
    headerTitle: {
        fontSize: 14,
        color: '#888',
        textTransform: 'uppercase',
        letterSpacing: 1,
        fontWeight: '600',
    },
    headerSubtitle: {
        fontSize: 28,
        fontWeight: '800',
        color: '#FFF',
        marginTop: 4,
    },
    content: {
        flex: 1,
    },
    scrollContent: {
        padding: 24,
        paddingBottom: 40,
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 24,
        alignItems: 'flex-start',
        maxWidth: 1600,
        width: '100%',
        alignSelf: 'center',
    },
    card: {
        backgroundColor: '#1E1E1E',
        borderRadius: 16,
        padding: 24,
        marginBottom: 0,
        borderWidth: 1,
        borderColor: '#2C2C2C',
        minWidth: 350,
        flex: 1,
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 20,
    },
    cardTitle: {
        fontSize: 20,
        fontWeight: '700',
        color: '#FFF',
        marginBottom: 12,
    },
    cardDescription: {
        fontSize: 14,
        color: '#AAA',
        marginBottom: 20,
        lineHeight: 20,
    },
    infoRow: {
        marginBottom: 16,
    },
    label: {
        fontSize: 12,
        color: '#888',
        marginBottom: 4,
        textTransform: 'uppercase',
    },
    value: {
        fontSize: 16,
        color: '#FFF',
    },
    pinContainer: {
        backgroundColor: '#000',
        padding: 10,
        borderRadius: 8,
        alignSelf: 'flex-start',
    },
    pinText: {
        fontSize: 24,
        color: '#FFF',
        fontWeight: 'bold',
        fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
        letterSpacing: 2,
    },
    statusBadge: {
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: 8,
    },
    statusText: {
        fontSize: 12,
        fontWeight: '700',
    },
    copyButton: {
        marginTop: 20,
        backgroundColor: '#333',
        paddingVertical: 14,
        borderRadius: 8,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#444',
    },
    copyButtonText: {
        color: '#FFF',
        fontWeight: '600',
        fontSize: 14,
    },
    actionButton: {
        paddingVertical: 16,
        borderRadius: 8,
        alignItems: 'center',
    },
    startButton: {
        backgroundColor: '#30D158',
    },
    stopButton: {
        backgroundColor: '#FF453A',
    },
    actionButtonText: {
        color: '#FFF',
        fontWeight: '700',
        fontSize: 16,
    },
    statsCard: {
    },
    statsGrid: {
        flexDirection: 'row',
        justifyContent: 'space-between',
    },
    statItem: {
        flex: 1,
        backgroundColor: '#121212',
        marginRight: 10,
        padding: 16,
        borderRadius: 12,
        alignItems: 'center',
    },
    statNumber: {
        fontSize: 24,
        fontWeight: '700',
        color: '#FFF',
        marginBottom: 4,
    },
    statLabel: {
        fontSize: 12,
        color: '#666',
    },
    qrCodeContainer: {
        alignItems: 'center',
        justifyContent: 'center',
        marginVertical: 10,
        padding: 10,
        backgroundColor: '#2C2C2C',
        borderRadius: 12,
    },
    qrPlaceholder: {
        height: 200,
        width: 200,
        backgroundColor: '#333',
        justifyContent: 'center',
        alignItems: 'center',
        borderRadius: 12,
    },
    qrPlaceholderText: {
        color: '#AAA',
        textAlign: 'center',
    },
    addButton: {
        backgroundColor: '#007AFF',
        paddingVertical: 8,
        paddingHorizontal: 16,
        borderRadius: 20,
    },
    addButtonText: {
        color: '#FFF',
        fontSize: 14,
        fontWeight: '600',
    },
    modulesList: {
        marginBottom: 20,
    },
    moduleItem: {
        backgroundColor: '#2C2C2C',
        borderRadius: 12,
        padding: 16,
        marginBottom: 12,
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    moduleInfo: {
        flex: 1,
    },
    moduleName: {
        fontSize: 16,
        fontWeight: '600',
        color: '#FFF',
        marginBottom: 4,
    },
    moduleType: {
        fontSize: 12,
        color: '#888',
        marginBottom: 8,
    },
    moduleStatusBadge: {
        paddingVertical: 4,
        paddingHorizontal: 8,
        borderRadius: 6,
        alignSelf: 'flex-start',
    },
    moduleStatusText: {
        fontSize: 10,
        fontWeight: '700',
    },
    moduleActions: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 10,
    },
    moduleBtn: {
        paddingVertical: 8,
        paddingHorizontal: 16,
        borderRadius: 8,
    },
    activateBtn: {
        backgroundColor: '#30D158',
    },
    endBtn: {
        backgroundColor: '#FF453A',
    },
    moduleBtnText: {
        color: '#FFF',
        fontSize: 14,
        fontWeight: '600',
    },
    endedText: {
        color: '#666',
        fontSize: 14,
        fontStyle: 'italic',
    },
    emptyModules: {
        padding: 40,
        alignItems: 'center',
        justifyContent: 'center',
        borderStyle: 'dashed',
        borderWidth: 1,
        borderColor: '#444',
        borderRadius: 12,
    },
    emptyModulesText: {
        color: '#666',
        fontSize: 16,
    },
    addModuleForm: {
        marginTop: 20,
        backgroundColor: '#252525',
        padding: 20,
        borderRadius: 12,
        borderWidth: 1,
        borderColor: '#333',
    },
    formTitle: {
        fontSize: 18,
        fontWeight: '700',
        color: '#FFF',
        marginBottom: 20,
    },
    typeSelector: {
        flexDirection: 'row',
        marginBottom: 20,
    },
    typeOption: {
        paddingVertical: 10,
        paddingHorizontal: 20,
        borderRadius: 8,
        backgroundColor: '#333',
        marginRight: 10,
    },
    selectedType: {
        backgroundColor: '#007AFF',
    },
    typeOptionText: {
        color: '#FFF',
        fontWeight: '600',
    },
    formActions: {
        flexDirection: 'row',
        justifyContent: 'flex-end',
        gap: 12,
        marginTop: 10,
    },
    cancelButton: {
        paddingVertical: 12,
        paddingHorizontal: 20,
        borderRadius: 8,
        backgroundColor: '#333',
    },
    cancelButtonText: {
        color: '#FFF',
        fontWeight: '600',
    },
    saveButton: {
        paddingVertical: 12,
        paddingHorizontal: 20,
        borderRadius: 8,
        backgroundColor: '#30D158',
    },
    saveButtonText: {
        color: '#FFF',
        fontWeight: '600',
    },
});
