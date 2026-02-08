import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, Animated, Clipboard, Alert, Platform, TextInput } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import QRCode from 'react-qr-code';
import { Ionicons } from '@expo/vector-icons';
import FormModuleConfig from './components/modules/FormModuleConfig';

const API_URL = 'http://localhost:8080';

export default function AdminEventDetailsScreen({ event, onBack }) {
    const [currentEvent, setCurrentEvent] = useState(event);
    const [loading, setLoading] = useState(!event);
    const [modules, setModules] = useState([]);

    // Original state for event status animation
    const [activeStatusAnim] = useState(new Animated.Value(event?.isActive ? 1 : 0));

    // Module Management State (original)
    const [moduleModalVisible, setModuleModalVisible] = useState(false);
    const [newModuleName, setNewModuleName] = useState('');
    const [selectedModuleType, setSelectedModuleType] = useState('form');

    // Timeline Editor State
    const [selectedModule, setSelectedModule] = useState(null);
    const [configModalVisible, setConfigModalVisible] = useState(false);


    useEffect(() => {
        if (event) {
            setCurrentEvent(event);
            setLoading(false);
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
            const response = await fetch(`${API_URL}/events/${currentEvent.id}`);
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
        if (!newModuleName.trim()) {
            Alert.alert('Error', 'Please enter a module name');
            return;
        }

        try {
            // Create a basic module. Configuration happens later.
            const moduleData = {
                type: selectedModuleType,
                name: newModuleName,
                data: {} // Empty data initially
            };

            const response = await fetch(`${API_URL}/events/${currentEvent.id}/modules`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(moduleData),
            });

            if (response.ok) {
                fetchEventDetails();
                setModuleModalVisible(false);
                setNewModuleName('');
                setSelectedModuleType('form');
                Alert.alert('Success', 'Module added. Now please configure it.');
            } else {
                const errText = await response.text();
                Alert.alert('Error', 'Failed to add module: ' + errText);
            }
        } catch (error) {
            console.error('Error adding module:', error);
            Alert.alert('Error', 'Failed to add module');
        }
    };



    const handleConfigureModule = (module) => {
        setSelectedModule(module);
        setConfigModalVisible(true);
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
                <View style={styles.headerTopRow}>
                    <TouchableOpacity onPress={onBack} style={styles.backLink}>
                        <Text style={styles.backLinkText}>← Back</Text>
                    </TouchableOpacity>
                    <Text style={styles.headerTitle}>Event Dashboard</Text>
                </View>
                <Text style={styles.headerSubtitle}>{currentEvent.name}</Text>
            </View>

            <ScrollView style={styles.content} contentContainerStyle={styles.scrollContent}>

                {/* Unified Dashboard Card */}
                <View style={[styles.card, styles.unifiedCard]}>
                    <View style={styles.cardHeader}>
                        <View>
                            <Text style={styles.cardTitle}>Event Overview</Text>
                            <Text style={styles.cardDescription}>Manage details, sharing, and controls.</Text>
                        </View>
                        <StatusIndicator />
                    </View>

                    <View style={styles.unifiedContent}>
                        {/* Left Column: Details & Controls */}
                        <View style={styles.leftColumn}>
                            <View style={styles.infoGrid}>
                                <View style={styles.infoItem}>
                                    <Text style={styles.label}>Description</Text>
                                    <Text style={styles.value}>{currentEvent.description}</Text>
                                </View>

                                <View style={styles.infoItem}>
                                    <Text style={styles.label}>PIN Code</Text>
                                    <View style={styles.pinContainer}>
                                        <Text style={styles.pinText}>{currentEvent.id}</Text>
                                    </View>
                                </View>
                            </View>

                            <View style={styles.controlSection}>
                                <Text style={styles.sectionLabel}>Event Controls</Text>
                                <TouchableOpacity
                                    style={[styles.actionButton, currentEvent.isActive ? styles.stopButton : styles.startButton]}
                                    onPress={handleToggleStatus}
                                >
                                    <Text style={styles.actionButtonText}>
                                        {currentEvent.isActive ? 'STOP EVENT' : 'START EVENT'}
                                    </Text>
                                </TouchableOpacity>
                                <Text style={styles.controlHint}>
                                    {currentEvent.isActive
                                        ? 'Event is live. Users can connect.'
                                        : 'Event is stopped. Users cannot connect.'}
                                </Text>
                            </View>
                        </View>

                        {/* Divider for desktop */}
                        <View style={styles.verticalDivider} />

                        {/* Right Column: Share Access */}
                        <View style={styles.rightColumn}>
                            <Text style={styles.sectionLabel}>Share Access</Text>
                            <View style={styles.qrCodeContainer}>
                                {Platform.OS === 'web' ? (
                                    <div style={{ background: 'white', padding: '12px', borderRadius: '12px' }}>
                                        <QRCode
                                            value={getShareLink()}
                                            size={160}
                                            style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                                            viewBox={`0 0 256 256`}
                                        />
                                    </div>
                                ) : (
                                    <View style={styles.qrPlaceholder}>
                                        <Text style={styles.qrPlaceholderText}>QR on Web</Text>
                                    </View>
                                )}
                            </View>
                            <TouchableOpacity style={styles.copyButton} onPress={copyToClipboard}>
                                <Text style={styles.copyButtonText}>Copy Link</Text>
                            </TouchableOpacity>
                        </View>
                    </View>
                </View>

                {/* 3. Modules Management Card */}
                <View style={[styles.card, { minWidth: 600, flex: 2 }]}>
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
                        Manage interactive modules for this event.
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
                                            <>
                                                <TouchableOpacity
                                                    style={[styles.moduleBtn, styles.configBtn]}
                                                    onPress={() => handleConfigureModule(module)}
                                                >
                                                    <Text style={styles.moduleBtnText}>Configure</Text>
                                                </TouchableOpacity>
                                                <TouchableOpacity
                                                    style={[styles.moduleBtn, styles.activateBtn]}
                                                    onPress={() => handleUpdateModuleStatus(module.id, 'active')}
                                                >
                                                    <Text style={styles.moduleBtnText}>Activate</Text>
                                                </TouchableOpacity>
                                            </>
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
                        <View style={styles.addModuleContainer}>
                            <View style={styles.addModuleHeader}>
                                <Text style={styles.formTitle}>Add Module</Text>
                                <TouchableOpacity onPress={() => setModuleModalVisible(false)}>
                                    <Text style={{ color: '#AAA', fontSize: 20 }}>✕</Text>
                                </TouchableOpacity>
                            </View>

                            <Text style={styles.inputLabel}>Module Name</Text>
                            <TextInput
                                style={styles.moduleInput}
                                placeholder="e.g. Feedback Form"
                                placeholderTextColor="#666"
                                value={newModuleName}
                                onChangeText={setNewModuleName}
                            />

                            <Text style={styles.inputLabel}>Module Type</Text>
                            <View style={styles.dropdownContainer}>
                                <TouchableOpacity style={styles.dropdown}>
                                    <Text style={styles.dropdownText}>📝 Form</Text>
                                    <Ionicons name="chevron-down" size={20} color="#888" />
                                </TouchableOpacity>
                            </View>

                            <TouchableOpacity style={styles.addModuleButton} onPress={handleAddModule}>
                                <Text style={styles.addModuleButtonText}>Add Module</Text>
                            </TouchableOpacity>
                        </View>
                    )}
                </View>

                {/* Stats Card - keeping separate as it might grow */}
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
            {configModalVisible && (
                <FormModuleConfig
                    visible={configModalVisible}
                    module={selectedModule}
                    eventId={currentEvent.id}
                    onClose={() => setConfigModalVisible(false)}
                    onUpdate={() => fetchEventDetails()}
                />
            )}
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
    },
    header: {
        paddingTop: 20, // Reduced from 60
        paddingBottom: 16, // Reduced from 20
        paddingHorizontal: 24,
        backgroundColor: '#1E1E1E',
        borderBottomWidth: 1,
        borderBottomColor: '#2C2C2C',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
    },
    headerTopRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: 4,
    },
    backLink: {
        // marginBottom: 10, removed
    },
    backLinkText: {
        color: '#007AFF',
        fontSize: 14,
    },
    headerTitle: {
        fontSize: 12,
        color: '#666',
        textTransform: 'uppercase',
        fontWeight: '600',
        letterSpacing: 1,
    },
    headerSubtitle: {
        fontSize: 20, // Reduced from 28
        fontWeight: '700',
        color: '#FFF',
    },
    content: {
        flex: 1,
    },
    unifiedCard: {
        minWidth: 600,
        flex: 3,
    },
    unifiedContent: {
        flexDirection: 'row',
        gap: 40,
        flexWrap: 'wrap',
    },
    leftColumn: {
        flex: 2,
        minWidth: 300,
    },
    rightColumn: {
        flex: 1,
        minWidth: 250,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#252525',
        borderRadius: 16,
        padding: 24,
    },
    verticalDivider: {
        width: 1,
        backgroundColor: '#333',
        height: '100%',
        display: Platform.OS === 'web' ? 'flex' : 'none',
    },
    infoGrid: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 20,
        marginBottom: 30,
    },
    infoItem: {
        minWidth: 150,
        flex: 1,
    },
    controlSection: {
        marginTop: 'auto',
        paddingTop: 20,
        borderTopWidth: 1,
        borderTopColor: '#2C2C2C',
    },
    sectionLabel: {
        fontSize: 14,
        fontWeight: '600',
        color: '#FFF',
        marginBottom: 16,
        textTransform: 'uppercase',
        letterSpacing: 1,
    },
    controlHint: {
        fontSize: 12,
        color: '#666',
        marginTop: 12,
        textAlign: 'center',
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
    // New Styles for Timeline Editor
    configBtn: {
        backgroundColor: '#FF9500', // Orange
    },
    modalContainer: {
        flex: 1,
        backgroundColor: '#121212',
    },
    // ... (rest of styles are fine, just need to bridge the gap)
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
    addModuleContainer: {
        marginTop: 20,
        backgroundColor: '#252525',
        padding: 24,
        borderRadius: 16,
        borderWidth: 1,
        borderColor: '#333',
    },
    addModuleHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 20,
    },
    moduleTypeList: {
        flexDirection: 'column',
        gap: 12,
        marginTop: 10,
    },
    moduleTypeListItem: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingVertical: 12,
        borderBottomWidth: 1,
        borderBottomColor: '#333',
    },
    moduleTypeListIcon: {
        fontSize: 20,
        marginRight: 10,
    },
    moduleTypeTitle: {
        color: '#FFF',
        fontSize: 16,
        fontWeight: '500',
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
    inputLabel: {
        fontSize: 14,
        fontWeight: '600',
        color: '#AAA',
        marginBottom: 8,
        marginTop: 16,
    },
    moduleInput: {
        backgroundColor: '#1E1E1E',
        borderRadius: 12,
        padding: 16,
        fontSize: 16,
        color: '#FFF',
        borderWidth: 1,
        borderColor: '#333',
    },
    dropdownContainer: {
        position: 'relative',
    },
    dropdown: {
        backgroundColor: '#1E1E1E',
        borderRadius: 12,
        padding: 16,
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#333',
    },
    dropdownText: {
        fontSize: 16,
        color: '#FFF',
    },
    addModuleButton: {
        backgroundColor: '#007AFF',
        paddingVertical: 16,
        borderRadius: 12,
        alignItems: 'center',
        marginTop: 24,
    },
    addModuleButtonText: {
        color: '#FFF',
        fontSize: 16,
        fontWeight: '600',
    },
});
